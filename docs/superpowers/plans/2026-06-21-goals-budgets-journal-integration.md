# Goals & Budgets: Editable + Journal Integration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add budget edit/delete UI, allow journal entries to tag multiple goals, and show live budget remaining inside the Record Entry modal.

**Architecture:** DB migration adds `journal_entry_goals` join table. Backend extends `JournalRepo` interface with `SaveGoalLinks` and updates `FindByUser` to populate `GoalIDs`. A new `DELETE /finance/budgets/:category` endpoint completes budget CRUD. Frontend replaces the BudgetsTab upsert card with a full management table, and the Record Entry modal gains goal multi-select + budget context panel.

**Tech Stack:** Go 1.22 + pgx/v5, React + Ant Design, React Query, TypeScript, Supabase migrations (SQL files in `supabase/migrations/`).

## Global Constraints

- All SQL migrations go in `supabase/migrations/` named `YYYYMMDDHHMMSS_<slug>.sql` (next after `20260221000001_account_asset_meta.sql`)
- Backend test coverage ≥ 80% per file in `internal/transport/http/` and `internal/middleware/`
- Run coverage check: `cd backend && bash scripts/hooks/pre-commit` from repo root
- Frontend: `cd frontend && npm run lint && npm run build` must be clean before PR
- JWT in memory only — never localStorage
- Currency default: `"VND"`
- All amounts stored/transmitted as strings (`decimal.Decimal` → `.String()`)

---

## File Map

| File | Change |
|------|--------|
| `supabase/migrations/20260621000002_journal_entry_goals.sql` | CREATE — new join table |
| `backend/internal/domain/accounting/journal_entry.go` | MODIFY — add `GoalIDs []string` field + `SetGoalIDs` / `GoalIDs()` accessors |
| `backend/internal/port/repository/accounting.go` | MODIFY — add `SaveGoalLinks` to `JournalRepo` interface; add `DeleteBudget` to `FinanceRepo` |
| `backend/internal/port/repository/finance.go` | MODIFY — add `DeleteBudget` |
| `backend/internal/infra/postgres/accounting_journal.go` | MODIFY — implement `SaveGoalLinks`, update `FindByUser` + `reconstituteEntries` |
| `backend/internal/infra/postgres/transactions.go` | MODIFY — implement `DeleteBudget` |
| `backend/internal/service/accounting/commands.go` | MODIFY — add `GoalIDs []string` to `RecordTransactionCmd` |
| `backend/internal/service/accounting/journal_service.go` | MODIFY — call `SaveGoalLinks` after save |
| `backend/internal/transport/http/accounting_journal.go` | MODIFY — decode/encode `goal_ids` |
| `backend/internal/transport/http/accounting_journal_test.go` | MODIFY — add goal_ids tests |
| `backend/internal/transport/http/transactions.go` | MODIFY — add `DeleteBudget` handler |
| `backend/internal/transport/http/transactions_test.go` | MODIFY — add delete budget test |
| `backend/cmd/server/main.go` | MODIFY — register `DELETE /budgets/{category}` route |
| `frontend/src/api/endpoints.ts` | MODIFY — add `deleteBudget`, update `createJournalEntry` + `getJournalEntries` types |
| `frontend/src/api/types.ts` | MODIFY — add `goal_ids` to `JournalEntry` and `CreateJournalEntryRequest` |
| `frontend/src/pages/WealthPage.tsx` | MODIFY — replace BudgetsTab with table + edit/delete |
| `frontend/src/pages/AccountingPage.tsx` | MODIFY — add goal multi-select + budget context to Record Entry modal; add Goals column to journal list |

---

### Task 1: DB Migration — `journal_entry_goals`

**Files:**
- Create: `supabase/migrations/20260621000002_journal_entry_goals.sql`

**Interfaces:**
- Produces: `journal_entry_goals(entry_id, goal_id, user_id)` table consumed by Task 3

- [ ] **Step 1: Create migration file**

```sql
-- supabase/migrations/20260621000002_journal_entry_goals.sql
CREATE TABLE journal_entry_goals (
  entry_id TEXT NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
  goal_id  TEXT NOT NULL REFERENCES goals(id)           ON DELETE CASCADE,
  user_id  TEXT NOT NULL,
  PRIMARY KEY (entry_id, goal_id)
);

CREATE INDEX idx_jeg_entry_id ON journal_entry_goals(entry_id);
CREATE INDEX idx_jeg_goal_id  ON journal_entry_goals(goal_id);
CREATE INDEX idx_jeg_user_id  ON journal_entry_goals(user_id);

ALTER TABLE journal_entry_goals ENABLE ROW LEVEL SECURITY;
CREATE POLICY "users own their journal_entry_goals"
  ON journal_entry_goals FOR ALL
  USING (user_id = auth.uid()::text);
```

- [ ] **Step 2: Apply migration locally**

```bash
docker compose up -d
supabase db push --local
```

Expected: migration applies without error.

- [ ] **Step 3: Commit**

```bash
git add supabase/migrations/20260621000002_journal_entry_goals.sql
git commit -m "chore(db): add journal_entry_goals join table"
```

---

### Task 2: Budget Delete — Backend

**Files:**
- Modify: `backend/internal/port/repository/finance.go`
- Modify: `backend/internal/infra/postgres/transactions.go`
- Modify: `backend/internal/transport/http/transactions.go`
- Modify: `backend/internal/transport/http/transactions_test.go`
- Modify: `backend/cmd/server/main.go`

**Interfaces:**
- Consumes: existing `TransactionHandler`, `FinanceRepo`
- Produces: `DELETE /finance/budgets/:category` → 204 No Content or 404

- [ ] **Step 1: Write the failing test**

In `backend/internal/transport/http/transactions_test.go`, add after the existing budget tests:

```go
func TestTransactionHandler_DeleteBudget(t *testing.T) {
	repo := &mockFinanceRepo{
		budgets: []finance.Budget{
			{ID: "b1", UserID: "user1", Category: "Food", MonthlyLimit: 500000},
		},
	}
	h := httphandler.NewTransactionHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/budgets/Food", nil)
	req = req.WithContext(withUserID(req.Context(), "user1"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, func() *chi.Context {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("category", "Food")
		return rctx
	}()))
	rr := httptest.NewRecorder()
	h.DeleteBudget(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d: %s", rr.Code, rr.Body.String())
	}
	if len(repo.budgets) != 0 {
		t.Errorf("expected budget to be deleted, still %d budgets", len(repo.budgets))
	}
}

func TestTransactionHandler_DeleteBudget_NotFound(t *testing.T) {
	repo := &mockFinanceRepo{budgets: []finance.Budget{}}
	h := httphandler.NewTransactionHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/budgets/Nonexistent", nil)
	req = req.WithContext(withUserID(req.Context(), "user1"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, func() *chi.Context {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("category", "Nonexistent")
		return rctx
	}()))
	rr := httptest.NewRecorder()
	h.DeleteBudget(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rr.Code)
	}
}
```

Also add `DeleteBudget` to the `mockFinanceRepo`:
```go
func (m *mockFinanceRepo) DeleteBudget(ctx context.Context, userID, category string) error {
	for i, b := range m.budgets {
		if b.UserID == userID && b.Category == category {
			m.budgets = append(m.budgets[:i], m.budgets[i+1:]...)
			return nil
		}
	}
	return repository.ErrBudgetNotFound
}
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
cd backend && go test ./internal/transport/http/... -run TestTransactionHandler_DeleteBudget -v
```

Expected: compile error — `DeleteBudget` undefined.

- [ ] **Step 3: Add `ErrBudgetNotFound` and `DeleteBudget` to repo interface**

In `backend/internal/port/repository/finance.go`:
```go
var ErrBudgetNotFound = errors.New("budget not found")

type FinanceRepo interface {
    // ... existing methods ...
    ListBudgets(ctx context.Context, userID string) ([]finance.Budget, error)
    UpsertBudget(ctx context.Context, b finance.Budget) (finance.Budget, error)
    DeleteBudget(ctx context.Context, userID, category string) error
    SumByUser(ctx context.Context, userID string) (float64, error)
    SumSpentThisMonth(ctx context.Context, userID string) (float64, error)
}
```

Add `"errors"` import if not present.

- [ ] **Step 4: Implement `DeleteBudget` in postgres**

In `backend/internal/infra/postgres/transactions.go`, add:
```go
func (r *pgFinanceRepo) DeleteBudget(ctx context.Context, userID, category string) error {
    tag, err := r.db.Exec(ctx,
        `DELETE FROM budgets WHERE user_id = $1 AND category = $2`,
        userID, category,
    )
    if err != nil {
        return err
    }
    if tag.RowsAffected() == 0 {
        return repository.ErrBudgetNotFound
    }
    return nil
}
```

- [ ] **Step 5: Add `DeleteBudget` HTTP handler**

In `backend/internal/transport/http/transactions.go`, add:
```go
func (h *TransactionHandler) DeleteBudget(w http.ResponseWriter, r *http.Request) {
    uid := middleware.GetUserID(r)
    category := chi.URLParam(r, "category")
    err := h.repo.DeleteBudget(r.Context(), uid, category)
    if errors.Is(err, repository.ErrBudgetNotFound) {
        http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
        return
    }
    if err != nil {
        http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}
```

Add `"errors"` import.

- [ ] **Step 6: Register route**

In `backend/cmd/server/main.go`, find the budget routes block and add:
```go
r.Delete("/budgets/{category}", txHandler.DeleteBudget)
```

- [ ] **Step 7: Run tests and verify pass**

```bash
cd backend && go test ./internal/transport/http/... -run TestTransactionHandler_DeleteBudget -v
```

Expected: PASS both tests.

- [ ] **Step 8: Run full coverage check**

```bash
cd backend && bash scripts/hooks/pre-commit
```

Expected: ✓ Coverage OK

- [ ] **Step 9: Commit**

```bash
git add backend/internal/port/repository/finance.go \
        backend/internal/infra/postgres/transactions.go \
        backend/internal/transport/http/transactions.go \
        backend/internal/transport/http/transactions_test.go \
        backend/cmd/server/main.go
git commit -m "feat(budget): add DELETE /finance/budgets/:category endpoint"
```

---

### Task 3: Journal Domain + Repo — Goal Links

**Files:**
- Modify: `backend/internal/domain/accounting/journal_entry.go`
- Modify: `backend/internal/port/repository/accounting.go`
- Modify: `backend/internal/infra/postgres/accounting_journal.go`
- Modify: `backend/internal/service/accounting/commands.go`
- Modify: `backend/internal/service/accounting/journal_service.go`

**Interfaces:**
- Produces:
  - `JournalEntry.GoalIDs() []string` — accessor
  - `JournalEntry.SetGoalIDs(ids []string)` — setter (used by repo on reconstitute)
  - `JournalRepo.SaveGoalLinks(ctx context.Context, entryID, userID string, goalIDs []string) error`
  - `RecordTransactionCmd.GoalIDs []string`
  - `FindByUser` returns entries with `GoalIDs` populated

- [ ] **Step 1: Write failing test for SaveGoalLinks**

In `backend/internal/transport/http/accounting_journal_test.go`, add:

```go
func TestJournalHandler_RecordTransaction_WithGoalIDs(t *testing.T) {
	jRepo := &testJournalRepo{}
	aRepo := newTestAccountRepoWithIDs("user1", "acc-food", "acc-visa")
	pub := &testPublisher{}

	journalSvc := accountingsvc.NewJournalService(jRepo, aRepo, pub)
	nwQuery := accountingsvc.NewNetWorthQuery(aRepo, jRepo)
	h := httphandler.NewJournalHandler(journalSvc, nwQuery)

	body, _ := json.Marshal(map[string]interface{}{
		"date":        "2026-07-01",
		"description": "Coffee",
		"goal_ids":    []string{"goal-1", "goal-2"},
		"lines": []map[string]interface{}{
			{"account_id": "acc-food", "amount": 150000, "side": "debit"},
			{"account_id": "acc-visa", "amount": 150000, "side": "credit"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/journal/entries", bytes.NewReader(body))
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr := httptest.NewRecorder()
	h.RecordTransaction(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if len(jRepo.goalLinks) == 0 {
		t.Error("expected goal links to be saved")
	}
}
```

Also update `testJournalRepo` to track goal links and implement the new interface:
```go
type testJournalRepo struct {
	saved     []*accounting.JournalEntry
	goalLinks map[string][]string // entryID -> goalIDs
}

func (r *testJournalRepo) Save(_ context.Context, e *accounting.JournalEntry) error {
	r.saved = append(r.saved, e)
	return nil
}

func (r *testJournalRepo) FindByUser(_ context.Context, _ string, _, _ time.Time) ([]*accounting.JournalEntry, error) {
	return r.saved, nil
}

func (r *testJournalRepo) SaveGoalLinks(_ context.Context, entryID, _ string, goalIDs []string) error {
	if r.goalLinks == nil {
		r.goalLinks = map[string][]string{}
	}
	r.goalLinks[entryID] = goalIDs
	return nil
}
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
cd backend && go test ./internal/transport/http/... -run TestJournalHandler_RecordTransaction_WithGoalIDs -v
```

Expected: compile error — `SaveGoalLinks` undefined.

- [ ] **Step 3: Add GoalIDs to JournalEntry domain struct**

In `backend/internal/domain/accounting/journal_entry.go`, add field and accessors:
```go
type JournalEntry struct {
    id          EntryID
    userID      string
    date        time.Time
    description string
    memo        string
    lines       []JournalLine
    events      []DomainEvent
    goalIDs     []string   // populated on reconstitution from DB
}

func (e *JournalEntry) GoalIDs() []string        { return slices.Clone(e.goalIDs) }
func (e *JournalEntry) SetGoalIDs(ids []string)  { e.goalIDs = ids }
```

- [ ] **Step 4: Add SaveGoalLinks to JournalRepo interface**

In `backend/internal/port/repository/accounting.go`:
```go
type JournalRepo interface {
    Save(ctx context.Context, e *accounting.JournalEntry) error
    FindByUser(ctx context.Context, userID string, from, to time.Time) ([]*accounting.JournalEntry, error)
    SaveGoalLinks(ctx context.Context, entryID, userID string, goalIDs []string) error
}
```

- [ ] **Step 5: Implement SaveGoalLinks in postgres**

In `backend/internal/infra/postgres/accounting_journal.go`, add:
```go
func (r *pgJournalRepo) SaveGoalLinks(ctx context.Context, entryID, userID string, goalIDs []string) error {
    if len(goalIDs) == 0 {
        return nil
    }
    batch := &pgx.Batch{}
    for _, gid := range goalIDs {
        batch.Queue(
            `INSERT INTO journal_entry_goals (entry_id, goal_id, user_id) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
            entryID, gid, userID,
        )
    }
    br := r.db.SendBatch(ctx, batch)
    defer br.Close()
    for range goalIDs {
        if _, err := br.Exec(); err != nil {
            // ignore FK violations (goal was deleted) — best-effort
            if !isFKViolation(err) {
                return err
            }
        }
    }
    return nil
}

// isFKViolation returns true for PostgreSQL error code 23503 (foreign_key_violation).
func isFKViolation(err error) bool {
    var pgErr *pgconn.PgError
    return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
```

Add imports: `"errors"`, `"github.com/jackc/pgx/v5/pgconn"`.

- [ ] **Step 6: Update FindByUser to populate GoalIDs**

Replace `reconstituteEntries` to also load goal links. Update `FindByUser` in `backend/internal/infra/postgres/accounting_journal.go`:

```go
func (r *pgJournalRepo) FindByUser(ctx context.Context, userID string, from, to time.Time) ([]*accounting.JournalEntry, error) {
    rows, err := r.db.Query(ctx, `
        SELECT e.id, e.user_id, e.date, e.description, e.memo,
               l.id, l.account_id, l.amount, l.currency, l.side
        FROM journal_entries e
        JOIN journal_lines l ON l.entry_id = e.id
        WHERE e.user_id = $1 AND ($2::date IS NULL OR e.date >= $2) AND ($3::date IS NULL OR e.date <= $3)
        ORDER BY e.date DESC, e.id, l.id`,
        userID, nullDate(from), nullDate(to),
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    entries, err := reconstituteEntries(rows)
    if err != nil {
        return nil, err
    }
    if len(entries) == 0 {
        return entries, nil
    }

    // load goal links for all entries in one query
    ids := make([]string, len(entries))
    for i, e := range entries {
        ids[i] = string(e.ID())
    }
    goalRows, err := r.db.Query(ctx,
        `SELECT entry_id, goal_id FROM journal_entry_goals WHERE entry_id = ANY($1)`,
        ids,
    )
    if err != nil {
        return nil, err
    }
    defer goalRows.Close()
    goalMap := map[string][]string{}
    for goalRows.Next() {
        var entryID, goalID string
        if err := goalRows.Scan(&entryID, &goalID); err != nil {
            return nil, err
        }
        goalMap[entryID] = append(goalMap[entryID], goalID)
    }
    for _, e := range entries {
        if gids, ok := goalMap[string(e.ID())]; ok {
            e.SetGoalIDs(gids)
        }
    }
    return entries, nil
}
```

- [ ] **Step 7: Add GoalIDs to RecordTransactionCmd and call SaveGoalLinks**

In `backend/internal/service/accounting/commands.go`:
```go
type RecordTransactionCmd struct {
    UserID      string
    Date        time.Time
    Description string
    Memo        string
    Lines       []LineCmd
    GoalIDs     []string
}
```

In `backend/internal/service/accounting/journal_service.go`, after `s.journal.Save(ctx, entry)`:
```go
if err := s.journal.Save(ctx, entry); err != nil {
    return "", err
}
if len(cmd.GoalIDs) > 0 {
    if err := s.journal.SaveGoalLinks(ctx, string(entry.ID()), cmd.UserID, cmd.GoalIDs); err != nil {
        return "", err
    }
}
for _, ev := range entry.Events() {
```

- [ ] **Step 8: Run test to verify pass**

```bash
cd backend && go test ./internal/transport/http/... -run TestJournalHandler_RecordTransaction_WithGoalIDs -v
```

Expected: PASS.

- [ ] **Step 9: Run full coverage check**

```bash
cd backend && bash scripts/hooks/pre-commit
```

Expected: ✓ Coverage OK

- [ ] **Step 10: Commit**

```bash
git add backend/internal/domain/accounting/journal_entry.go \
        backend/internal/port/repository/accounting.go \
        backend/internal/infra/postgres/accounting_journal.go \
        backend/internal/service/accounting/commands.go \
        backend/internal/service/accounting/journal_service.go \
        backend/internal/transport/http/accounting_journal_test.go
git commit -m "feat(journal): add goal tagging — SaveGoalLinks + GoalIDs on entries"
```

---

### Task 4: Journal HTTP Handler — goal_ids in/out

**Files:**
- Modify: `backend/internal/transport/http/accounting_journal.go`
- Modify: `backend/internal/transport/http/accounting_journal_test.go`

**Interfaces:**
- Consumes: `RecordTransactionCmd.GoalIDs`, `JournalEntry.GoalIDs()`
- Produces: `POST /accounting/journal` accepts `"goal_ids": ["..."]`; `GET /accounting/journal` returns `"goal_ids": ["..."]`

- [ ] **Step 1: Write failing test for ListEntries returning goal_ids**

In `backend/internal/transport/http/accounting_journal_test.go`, add:

```go
func TestJournalHandler_ListEntries_IncludesGoalIDs(t *testing.T) {
	entry := accounting.ReconstituteEntry("e1", "user1", time.Now(), "desc", "")
	entry.SetGoalIDs([]string{"g1", "g2"})
	entry.ReconstituteLine("l1", "acc1", accounting.Money{Amount: decimal.NewFromInt(1), Currency: "VND"}, accounting.Debit)
	entry.ReconstituteLine("l2", "acc2", accounting.Money{Amount: decimal.NewFromInt(1), Currency: "VND"}, accounting.Credit)

	jRepo := &testJournalRepo{saved: []*accounting.JournalEntry{entry}}
	aRepo := newTestAccountRepoWithIDs("user1", "acc1", "acc2")
	pub := &testPublisher{}

	journalSvc := accountingsvc.NewJournalService(jRepo, aRepo, pub)
	nwQuery := accountingsvc.NewNetWorthQuery(aRepo, jRepo)
	h := httphandler.NewJournalHandler(journalSvc, nwQuery)

	req := httptest.NewRequest(http.MethodGet, "/api/journal/entries", nil)
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr := httptest.NewRecorder()
	h.ListEntries(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var result []map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)
	if len(result) == 0 {
		t.Fatal("expected entries")
	}
	goalIDs, ok := result[0]["goal_ids"].([]interface{})
	if !ok || len(goalIDs) != 2 {
		t.Errorf("expected 2 goal_ids, got %v", result[0]["goal_ids"])
	}
}
```

- [ ] **Step 2: Run to confirm fail**

```bash
cd backend && go test ./internal/transport/http/... -run TestJournalHandler_ListEntries_IncludesGoalIDs -v
```

Expected: FAIL — `goal_ids` not in response.

- [ ] **Step 3: Update RecordTransaction handler to decode goal_ids**

In `backend/internal/transport/http/accounting_journal.go`, update the request struct in `RecordTransaction`:
```go
var req struct {
    Date        string   `json:"date"`
    Description string   `json:"description"`
    Memo        string   `json:"memo"`
    GoalIDs     []string `json:"goal_ids"`
    Lines       []struct {
        AccountID string          `json:"account_id"`
        Amount    decimal.Decimal `json:"amount"`
        Currency  string          `json:"currency"`
        Side      string          `json:"side"`
    } `json:"lines"`
}
```

And pass to cmd:
```go
cmd := accountingsvc.RecordTransactionCmd{
    UserID:      userID,
    Date:        date,
    Description: req.Description,
    Memo:        req.Memo,
    Lines:       lines,
    GoalIDs:     req.GoalIDs,
}
```

- [ ] **Step 4: Update ListEntries handler to include goal_ids**

In `backend/internal/transport/http/accounting_journal.go`, update the `entryResp` struct and its construction in `ListEntries`:

```go
type entryResp struct {
    ID          string     `json:"id"`
    Date        string     `json:"date"`
    Description string     `json:"description"`
    Memo        string     `json:"memo"`
    Lines       []lineResp `json:"lines"`
    GoalIDs     []string   `json:"goal_ids"`
}
```

And in the loop:
```go
goalIDs := e.GoalIDs()
if goalIDs == nil {
    goalIDs = []string{}
}
result = append(result, entryResp{
    ID:          string(e.ID()),
    Date:        e.Date().Format("2006-01-02"),
    Description: e.Description(),
    Memo:        e.Memo(),
    Lines:       lines,
    GoalIDs:     goalIDs,
})
```

- [ ] **Step 5: Run tests to verify pass**

```bash
cd backend && go test ./internal/transport/http/... -run TestJournalHandler -v
```

Expected: all journal handler tests PASS.

- [ ] **Step 6: Run full coverage check**

```bash
cd backend && bash scripts/hooks/pre-commit
```

Expected: ✓ Coverage OK

- [ ] **Step 7: Commit**

```bash
git add backend/internal/transport/http/accounting_journal.go \
        backend/internal/transport/http/accounting_journal_test.go
git commit -m "feat(journal): decode/encode goal_ids in HTTP handler"
```

---

### Task 5: Budget Management UI (WealthPage)

**Files:**
- Modify: `frontend/src/api/endpoints.ts`
- Modify: `frontend/src/pages/WealthPage.tsx`

**Interfaces:**
- Consumes: `GET /finance/budgets`, `PUT /finance/budgets/:category`, `DELETE /finance/budgets/:category`
- Produces: BudgetsTab with table, Edit modal, Delete confirmation

- [ ] **Step 1: Add deleteBudget to endpoints.ts**

In `frontend/src/api/endpoints.ts`, add after `upsertBudget`:
```ts
export const deleteBudget = (category: string) =>
  apiClient.delete(`/budgets/${category}`)
```

- [ ] **Step 2: Replace BudgetsTab in WealthPage.tsx**

Find the `function BudgetsTab()` in `frontend/src/pages/WealthPage.tsx` and replace its entire body:

```tsx
function BudgetsTab() {
  const [editBudget, setEditBudget] = useState<Budget | null>(null)
  const [addOpen, setAddOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: txs = [] } = useQuery({ queryKey: ['transactions'], queryFn: () => getTransactions() })
  const { data: budgets = [] } = useQuery({ queryKey: ['budgets'], queryFn: getBudgets })

  const upsertMutation = useMutation({
    mutationFn: ({ category, monthly_limit }: { category: string; monthly_limit: number }) =>
      upsertBudget(category, monthly_limit),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['budgets'] })
      setEditBudget(null)
      setAddOpen(false)
      form.resetFields()
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (category: string) => deleteBudget(category),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['budgets'] }),
  })

  const openEdit = (b: Budget) => {
    setEditBudget(b)
    form.setFieldsValue({ monthly_limit: b.monthly_limit })
  }

  const openAdd = () => {
    setAddOpen(true)
    form.resetFields()
  }

  return (
    <>
      {budgets.length > 0 && (
        <Card size="small" title="Budget Progress" style={{ marginBottom: 12 }}>
          <Row gutter={[12, 8]}>
            {budgets.map(b => {
              const spent = txs.filter(t => t.category === b.category && t.amount < 0).reduce((s, t) => s + Math.abs(t.amount), 0)
              const pct = Math.min(Math.round(spent / b.monthly_limit * 100), 100)
              return (
                <Col xs={24} sm={8} key={b.id}>
                  <div style={{ fontSize: 12, marginBottom: 2 }}>{b.category} <span style={{ color: '#999' }}>{fmtVND(spent)} / {fmtVND(b.monthly_limit)}</span></div>
                  <Progress percent={pct} size="small" strokeColor={pct > 90 ? '#ff4d4f' : '#1677ff'} />
                </Col>
              )
            })}
          </Row>
        </Card>
      )}

      <Card
        size="small"
        title="Budgets"
        extra={<Button size="small" type="primary" icon={<PlusOutlined />} onClick={openAdd}>Add Budget</Button>}
      >
        <Table<Budget>
          dataSource={budgets}
          rowKey="id"
          size="small"
          pagination={false}
          columns={[
            { title: 'Category', dataIndex: 'category' },
            { title: 'Monthly Limit', dataIndex: 'monthly_limit', render: (v: number) => fmtVND(v) },
            {
              title: 'Actions',
              width: 100,
              render: (_: unknown, b: Budget) => (
                <Space>
                  <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(b)} />
                  <Popconfirm
                    title={`Delete budget for ${b.category}?`}
                    onConfirm={() => deleteMutation.mutate(b.category)}
                    okText="Delete"
                    okButtonProps={{ danger: true }}
                  >
                    <Button size="small" danger icon={<DeleteOutlined />} loading={deleteMutation.isPending} />
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
          locale={{ emptyText: 'No budgets yet.' }}
        />
      </Card>

      {/* Edit modal */}
      <Modal
        title={`Edit Budget — ${editBudget?.category}`}
        open={!!editBudget}
        onCancel={() => { setEditBudget(null); form.resetFields() }}
        footer={null}
      >
        <Form form={form} layout="vertical" onFinish={v => upsertMutation.mutate({ category: editBudget!.category, monthly_limit: v.monthly_limit })}>
          <Form.Item name="monthly_limit" label="Monthly Limit (₫)" rules={[{ required: true }]}>
            <InputNumber min={0} step={1} style={{ width: '100%' }} />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={upsertMutation.isPending} block>Save</Button>
        </Form>
      </Modal>

      {/* Add modal */}
      <Modal
        title="Add Budget"
        open={addOpen}
        onCancel={() => { setAddOpen(false); form.resetFields() }}
        footer={null}
      >
        <Form form={form} layout="vertical" onFinish={v => upsertMutation.mutate(v)}>
          <Form.Item name="category" label="Category" rules={[{ required: true }]}>
            <Select placeholder="Select category" options={CATEGORIES.map(c => ({ value: c, label: c }))} />
          </Form.Item>
          <Form.Item name="monthly_limit" label="Monthly Limit (₫)" rules={[{ required: true }]}>
            <InputNumber min={0} step={1} style={{ width: '100%' }} />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={upsertMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </>
  )
}
```

Ensure `Popconfirm` is imported from `antd`, and `deleteBudget` is imported from `../api/endpoints`. Add `Table` to antd imports if not present.

- [ ] **Step 3: Run lint + build**

```bash
cd frontend && npm run lint && npm run build
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/api/endpoints.ts frontend/src/pages/WealthPage.tsx
git commit -m "feat(budget): full budget management UI — edit modal + delete"
```

---

### Task 6: Journal UI — Goal Tags + Budget Context

**Files:**
- Modify: `frontend/src/api/types.ts`
- Modify: `frontend/src/pages/AccountingPage.tsx`

**Interfaces:**
- Consumes:
  - `GET /accounting/journal` entries now include `goal_ids: string[]`
  - `POST /accounting/journal` accepts `goal_ids: string[]`
  - `['goals']` query (already used in ObjectivesPage — same cache key, same `Goal` type)
  - `['budgets']` + `['transactions']` queries (already in WealthPage, available globally)
- Produces:
  - Record Entry modal: goal multi-select + budget remaining panel
  - Journal list: Goals column

- [ ] **Step 1: Update API types**

In `frontend/src/api/types.ts`:

Find `JournalEntry` and add `goal_ids`:
```ts
export interface JournalEntry {
  id: string
  date: string
  description: string
  memo: string
  lines: {
    id: string
    account_id: string
    amount: string
    currency: string
    side: 'debit' | 'credit'
  }[]
  goal_ids: string[]
}
```

Find `CreateJournalEntryRequest` and add `goal_ids`:
```ts
export interface CreateJournalEntryRequest {
  date: string
  description: string
  memo: string
  lines: {
    account_id: string
    amount: string
    currency: string
    side: 'debit' | 'credit'
  }[]
  goal_ids?: string[]
}
```

- [ ] **Step 2: Add getGoals import to AccountingPage**

In `frontend/src/pages/AccountingPage.tsx`, add to the imports from `../api/endpoints`:
```ts
import { getAccounts, createAccount, updateAccount, deleteAccount, createJournalEntry, getJournalEntries, getJournalNetWorth, getGoals } from '../api/endpoints'
```

Add `Goal` to type imports:
```ts
import type { Account, CreateAccountRequest, UpdateAccountRequest, CreateJournalEntryRequest, JournalEntry, Goal } from '../api/types'
```

- [ ] **Step 3: Update JournalTab — add goals query, budget query, and update Record Entry modal**

In `frontend/src/pages/AccountingPage.tsx`, inside `function JournalTab()`, add queries after the existing ones:

```tsx
const { data: goals = [] } = useQuery({ queryKey: ['goals'], queryFn: getGoals })
const { data: budgets = [] } = useQuery({ queryKey: ['budgets'], queryFn: getBudgets })
const { data: txs = [] } = useQuery({ queryKey: ['transactions'], queryFn: () => getTransactions() })
```

Add `getBudgets` and `getTransactions` to the endpoints import (they're already exported from `endpoints.ts`).

Update `recordMutation` `mutationFn` to include `goal_ids`:
```tsx
const recordMutation = useMutation({
  mutationFn: (values: {
    date: string
    description: string
    memo: string
    goal_ids?: string[]
    lines: { account_id: string; amount: number; side: 'debit' | 'credit' }[]
  }) => {
    const req: CreateJournalEntryRequest = {
      date: values.date,
      description: values.description,
      memo: values.memo ?? '',
      goal_ids: values.goal_ids ?? [],
      lines: values.lines.map(l => ({
        account_id: l.account_id,
        amount: String(l.amount),
        currency: 'VND',
        side: l.side,
      })),
    }
    return createJournalEntry(req)
  },
  // ...existing onSuccess
})
```

- [ ] **Step 4: Add goal_ids field to Record Entry modal form**

Inside the `<Modal title="Record Journal Entry" ...>` form, add this `Form.Item` before the submit button:

```tsx
<Form.Item name="goal_ids" label="Goals (optional)">
  <Select
    mode="multiple"
    placeholder="Tag with goals"
    options={goals.map(g => ({
      value: g.id,
      label: (
        <span>
          <span style={{ display: 'inline-block', width: 10, height: 10, borderRadius: '50%', background: g.color, marginRight: 6 }} />
          {g.name}
        </span>
      ),
    }))}
    filterOption={(input, opt) =>
      (goals.find(g => g.id === opt?.value)?.name ?? '').toLowerCase().includes(input.toLowerCase())
    }
  />
</Form.Item>
```

- [ ] **Step 5: Add budget context panel to Record Entry modal**

Add a budget context component inside the modal, after the lines form items and before the submit button:

```tsx
{budgets.length > 0 && (() => {
  const now = new Date()
  const monthStart = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-01`
  const monthEnd = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate()}`
  return (
    <div style={{ marginBottom: 12 }}>
      <div style={{ fontSize: 12, color: '#8c8c8c', marginBottom: 4 }}>Budget remaining this month</div>
      <Space wrap size={[4, 4]}>
        {budgets.map(b => {
          const spent = txs
            .filter(t => {
              const d = t.date?.slice(0, 10) ?? ''
              return t.category === b.category && t.amount < 0 && d >= monthStart && d <= monthEnd
            })
            .reduce((s, t) => s + Math.abs(t.amount), 0)
          const remaining = b.monthly_limit - spent
          const pct = b.monthly_limit > 0 ? remaining / b.monthly_limit : 1
          const color = pct <= 0.2 ? '#ff4d4f' : '#52c41a'
          return (
            <Tag key={b.id} color={pct <= 0.2 ? 'red' : 'green'} style={{ fontSize: 11 }}>
              {b.category}: <b style={{ color }}>{fmtVND(String(remaining))}</b>
            </Tag>
          )
        })}
      </Space>
    </div>
  )
})()}
```

- [ ] **Step 6: Add Goals column to journal table**

In the `<Table<JournalEntry>` columns array, add after the Lines column:

```tsx
{
  title: 'Goals',
  dataIndex: 'goal_ids',
  render: (gids: string[]) => {
    if (!gids?.length) return null
    return (
      <Space wrap size={[4, 4]}>
        {gids.map(gid => {
          const g = goals.find(x => x.id === gid)
          if (!g) return null
          return (
            <Tag key={gid} style={{ fontSize: 11, borderColor: g.color, color: g.color, background: `${g.color}18` }}>
              {g.name}
            </Tag>
          )
        })}
      </Space>
    )
  },
},
```

- [ ] **Step 7: Run lint + build**

```bash
cd frontend && npm run lint && npm run build
```

Expected: no errors.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/api/types.ts frontend/src/pages/AccountingPage.tsx
git commit -m "feat(journal): goal tagging + budget context in Record Entry modal"
```

---

### Task 7: Integration verification + PR

- [ ] **Step 1: Run all backend tests with coverage**

```bash
cd backend && bash scripts/hooks/pre-commit
```

Expected: ✓ Coverage OK, all files ≥ 80%

- [ ] **Step 2: Run frontend lint + build**

```bash
cd frontend && npm run lint && npm run build
```

Expected: clean

- [ ] **Step 3: Run integration smoke tests**

```bash
bash scripts/integration-test.sh
```

Expected: PASS

- [ ] **Step 4: Create PR**

```bash
git push -u origin <branch>
gh pr create --title "feat: goals & budgets editable + journal integration" --body "$(cat <<'EOF'
## Summary
- Budget management: replace upsert-only form with full table (edit modal + delete per row)
- Journal entries can be tagged with multiple goals via multi-select
- Record Entry modal shows live budget remaining per category for current month

## Migration
`20260621000002_journal_entry_goals.sql` — adds `journal_entry_goals` join table

## Test plan
- [ ] Add a budget, edit its monthly limit via modal, verify updated
- [ ] Delete a budget, verify it disappears from table
- [ ] Record a journal entry with goal tags, verify tags appear in journal list
- [ ] Open Record Entry modal, verify budget remaining panel shows correct amounts
- [ ] Record entry that would exhaust a budget, verify tag turns red
EOF
)"
gh pr merge --auto --squash
```

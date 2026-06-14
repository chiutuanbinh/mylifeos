# Liabilities Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `liabilities` data model with CRUD API and UI tab so users can track debts; net worth = assets − liabilities.

**Architecture:** New `liabilities` table, `LiabilityRepo`/`LiabilityHandler` pair mirroring the existing assets pattern, registered in `main.go`. Frontend adds a "Liabilities" tab to `WealthPage` and updates the net worth summary widget. The `TrendsHandler` receives a `LiabilityRepo` so its snapshot helper can subtract total liabilities from assets.

**Tech Stack:** Go + pgx/v5 + chi (backend), React + Ant Design + TanStack Query (frontend), PostgreSQL migrations (Supabase + embedded)

---

## File Map

| Action | Path | Responsibility |
|--------|------|----------------|
| Create | `backend/internal/migrate/006_liabilities.sql` | DDL for liabilities table |
| Create | `supabase/migrations/20260614000006_liabilities.sql` | Same DDL for Supabase |
| Modify | `backend/internal/models/models.go` | Add `Liability` struct |
| Create | `backend/internal/repo/liabilities.go` | `LiabilityRepo` interface + pgx impl |
| Create | `backend/internal/handlers/liabilities.go` | CRUD handler |
| Create | `backend/internal/handlers/liabilities_test.go` | Handler unit tests |
| Modify | `backend/internal/handlers/trends.go` | Accept `LiabilityRepo`, subtract in net-worth helper |
| Modify | `backend/internal/handlers/trends_test.go` | Add mock for `LiabilityRepo` |
| Modify | `backend/cmd/server/main.go` | Wire up new repo + handler + routes |
| Modify | `frontend/src/api/types.ts` | Add `Liability` type |
| Modify | `frontend/src/api/endpoints.ts` | Add liability CRUD endpoints |
| Modify | `frontend/src/pages/WealthPage.tsx` | Add `LiabilitiesTab` + update summary widget |

---

## Task 1: DB Migrations

**Files:**
- Create: `backend/internal/migrate/006_liabilities.sql`
- Create: `supabase/migrations/20260614000006_liabilities.sql`

- [ ] **Step 1: Write backend embedded migration**

```sql
-- backend/internal/migrate/006_liabilities.sql
CREATE TABLE IF NOT EXISTS public.liabilities (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id            TEXT NOT NULL,
  name               TEXT NOT NULL,
  category           TEXT NOT NULL,
  balance            FLOAT8 NOT NULL DEFAULT 0,
  original_principal FLOAT8,
  interest_rate      FLOAT8,
  started_at         DATE,
  due_at             DATE,
  notes              TEXT NOT NULL DEFAULT '',
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS liabilities_user_id_idx ON public.liabilities(user_id);
```

- [ ] **Step 2: Write Supabase migration (identical SQL)**

Copy the same SQL to `supabase/migrations/20260614000006_liabilities.sql`.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/migrate/006_liabilities.sql supabase/migrations/20260614000006_liabilities.sql
git commit -m "chore: add liabilities table migration"
```

---

## Task 2: Liability Model

**Files:**
- Modify: `backend/internal/models/models.go`

- [ ] **Step 1: Add `Liability` struct after the `Asset` struct**

In `backend/internal/models/models.go`, add after the closing brace of `Asset`:

```go
type Liability struct {
	ID                string   `json:"id"`
	UserID            string   `json:"user_id"`
	Name              string   `json:"name"`
	Category          string   `json:"category"`
	Balance           float64  `json:"balance"`
	OriginalPrincipal *float64 `json:"original_principal"`
	InterestRate      *float64 `json:"interest_rate"`
	StartedAt         *string  `json:"started_at"`
	DueAt             *string  `json:"due_at"`
	Notes             string   `json:"notes"`
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd backend && go build ./...
```

Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add backend/internal/models/models.go
git commit -m "feat: add Liability model"
```

---

## Task 3: LiabilityRepo

**Files:**
- Create: `backend/internal/repo/liabilities.go`

- [ ] **Step 1: Write the repo**

Create `backend/internal/repo/liabilities.go`:

```go
package repo

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LiabilityRepo interface {
	List(ctx context.Context, userID string) ([]models.Liability, error)
	Create(ctx context.Context, l models.Liability) (models.Liability, error)
	Update(ctx context.Context, l models.Liability) (models.Liability, error)
	Delete(ctx context.Context, id, userID string) error
	TotalBalance(ctx context.Context, userID string) (float64, error)
}

type pgLiabilityRepo struct{ db *pgxpool.Pool }

func NewLiabilityRepo(db *pgxpool.Pool) LiabilityRepo { return &pgLiabilityRepo{db} }

func scanLiability(row interface{ Scan(...any) error }) (models.Liability, error) {
	var l models.Liability
	var startedAt, dueAt *time.Time
	err := row.Scan(&l.ID, &l.UserID, &l.Name, &l.Category, &l.Balance,
		&l.OriginalPrincipal, &l.InterestRate, &startedAt, &dueAt, &l.Notes)
	if startedAt != nil {
		s := startedAt.Format("2006-01-02")
		l.StartedAt = &s
	}
	if dueAt != nil {
		s := dueAt.Format("2006-01-02")
		l.DueAt = &s
	}
	return l, err
}

const liabilityCols = `id, user_id, name, category, balance, original_principal, interest_rate, started_at, due_at, notes`

func (r *pgLiabilityRepo) List(ctx context.Context, userID string) ([]models.Liability, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+liabilityCols+` FROM liabilities WHERE user_id=$1 ORDER BY category, name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Liability
	for rows.Next() {
		l, err := scanLiability(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	if out == nil {
		out = []models.Liability{}
	}
	return out, rows.Err()
}

func (r *pgLiabilityRepo) Create(ctx context.Context, l models.Liability) (models.Liability, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO liabilities (user_id, name, category, balance, original_principal, interest_rate, started_at, due_at, notes)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING `+liabilityCols,
		l.UserID, l.Name, l.Category, l.Balance, l.OriginalPrincipal, l.InterestRate, l.StartedAt, l.DueAt, l.Notes)
	return scanLiability(row)
}

func (r *pgLiabilityRepo) Update(ctx context.Context, l models.Liability) (models.Liability, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE liabilities SET name=$1, category=$2, balance=$3, original_principal=$4,
		 interest_rate=$5, started_at=$6, due_at=$7, notes=$8
		 WHERE id=$9 AND user_id=$10
		 RETURNING `+liabilityCols,
		l.Name, l.Category, l.Balance, l.OriginalPrincipal, l.InterestRate, l.StartedAt, l.DueAt, l.Notes, l.ID, l.UserID)
	return scanLiability(row)
}

func (r *pgLiabilityRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM liabilities WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

func (r *pgLiabilityRepo) TotalBalance(ctx context.Context, userID string) (float64, error) {
	var total float64
	err := r.db.QueryRow(ctx, `SELECT COALESCE(SUM(balance),0) FROM liabilities WHERE user_id=$1`, userID).Scan(&total)
	return total, err
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd backend && go build ./...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/repo/liabilities.go
git commit -m "feat: add LiabilityRepo"
```

---

## Task 4: LiabilityHandler + Tests

**Files:**
- Create: `backend/internal/handlers/liabilities.go`
- Create: `backend/internal/handlers/liabilities_test.go`

- [ ] **Step 1: Write failing tests first**

Create `backend/internal/handlers/liabilities_test.go`:

```go
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/chiutuanbinh/mylifeos/backend/internal/handlers"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
)

type mockLiabilityRepo struct{}

func (m *mockLiabilityRepo) List(_ context.Context, _ string) ([]models.Liability, error) {
	ir := 0.085
	return []models.Liability{{ID: "l-1", Name: "Car Loan", Category: "Car Loan", Balance: 200000000, InterestRate: &ir}}, nil
}
func (m *mockLiabilityRepo) Create(_ context.Context, l models.Liability) (models.Liability, error) {
	l.ID = "l-new"
	return l, nil
}
func (m *mockLiabilityRepo) Update(_ context.Context, l models.Liability) (models.Liability, error) {
	return l, nil
}
func (m *mockLiabilityRepo) Delete(_ context.Context, _, _ string) error { return nil }
func (m *mockLiabilityRepo) TotalBalance(_ context.Context, _ string) (float64, error) {
	return 200000000, nil
}

func TestLiabilityList(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.List))

	req := httptest.NewRequest("GET", "/api/v1/liabilities", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var items []models.Liability
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) != 1 || items[0].ID != "l-1" {
		t.Fatalf("unexpected: %+v", items)
	}
}

func TestLiabilityCreate(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(map[string]any{"name": "Mortgage", "category": "Mortgage", "balance": 500000000.0})
	req := httptest.NewRequest("POST", "/api/v1/liabilities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLiabilityCreate_MissingName(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(map[string]any{"category": "Mortgage", "balance": 500000000.0})
	req := httptest.NewRequest("POST", "/api/v1/liabilities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLiabilityCreate_NegativeBalance(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(map[string]any{"name": "Loan", "category": "Personal Loan", "balance": -1000.0})
	req := httptest.NewRequest("POST", "/api/v1/liabilities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLiabilityCreate_InvalidInterestRate(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	ir := 1.5
	body, _ := json.Marshal(map[string]any{"name": "Loan", "category": "Personal Loan", "balance": 1000.0, "interest_rate": ir})
	req := httptest.NewRequest("POST", "/api/v1/liabilities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLiabilityCreate_BadJSON(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	req := httptest.NewRequest("POST", "/api/v1/liabilities", bytes.NewReader([]byte("notjson")))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLiabilityUpdate(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepo{})
	r := chi.NewRouter()
	r.Patch("/liabilities/{id}", middleware.Auth(http.HandlerFunc(h.Update)).(http.HandlerFunc))

	body, _ := json.Marshal(map[string]any{"name": "Updated Loan", "category": "Car Loan", "balance": 180000000.0})
	req := httptest.NewRequest("PATCH", "/liabilities/l-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestLiabilityDelete(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepo{})
	r := chi.NewRouter()
	r.Delete("/liabilities/{id}", middleware.Auth(http.HandlerFunc(h.Delete)).(http.HandlerFunc))

	req := httptest.NewRequest("DELETE", "/liabilities/l-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run tests — expect compile failure (handler not yet written)**

```bash
cd backend && go test ./internal/handlers/... 2>&1 | head -20
```

Expected: `undefined: handlers.NewLiabilityHandler`

- [ ] **Step 3: Write handler**

Create `backend/internal/handlers/liabilities.go`:

```go
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/chiutuanbinh/mylifeos/backend/internal/repo"
)

type LiabilityHandler struct{ repo repo.LiabilityRepo }

func NewLiabilityHandler(r repo.LiabilityRepo) *LiabilityHandler { return &LiabilityHandler{r} }

func validateLiability(l models.Liability) string {
	if l.Name == "" {
		return "name is required"
	}
	if l.Category == "" {
		return "category is required"
	}
	if l.Balance < 0 {
		return "balance must be >= 0"
	}
	if l.InterestRate != nil && (*l.InterestRate < 0 || *l.InterestRate > 1) {
		return "interest_rate must be between 0 and 1"
	}
	return ""
}

func (h *LiabilityHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	items, err := h.repo.List(r.Context(), uid)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *LiabilityHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var l models.Liability
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	if msg := validateLiability(l); msg != "" {
		http.Error(w, `{"error":"`+msg+`"}`, 400)
		return
	}
	l.UserID = uid
	out, err := h.repo.Create(r.Context(), l)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(out)
}

func (h *LiabilityHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var l models.Liability
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	if msg := validateLiability(l); msg != "" {
		http.Error(w, `{"error":"`+msg+`"}`, 400)
		return
	}
	l.ID = chi.URLParam(r, "id")
	l.UserID = uid
	out, err := h.repo.Update(r.Context(), l)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *LiabilityHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	if err := h.repo.Delete(r.Context(), chi.URLParam(r, "id"), uid); err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.WriteHeader(204)
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
cd backend && go test ./internal/handlers/... -coverprofile=coverage.out -covermode=atomic && bash scripts/hooks/pre-commit
```

Expected: all tests pass, per-file coverage ≥ 80%.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/handlers/liabilities.go backend/internal/handlers/liabilities_test.go
git commit -m "feat: add LiabilityHandler with CRUD and validation"
```

---

## Task 5: Wire TrendsHandler to LiabilityRepo

The `TrendsHandler.AddSnapshot` should subtract total liabilities from assets value when persisting net worth. Currently it accepts `net_worth` directly from the client — we keep that (client controls the snapshot value) but the existing `assetRepo` reference pattern shows where to add the liability context if needed in future. For now: no change to `TrendsHandler` logic is required because the frontend will compute `net_worth = assets - liabilities` before posting the snapshot.

However, the `TrendsHandler` tests use a `mockTrendsRepo` that must still compile after the `LiabilityRepo` addition in `main.go`. No changes needed to `trends.go` or `trends_test.go`.

- [ ] **Step 1: Verify trends tests still pass**

```bash
cd backend && go test ./internal/handlers/... -v 2>&1 | grep -E "PASS|FAIL|ok"
```

Expected: all PASS.

---

## Task 6: Register Routes in main.go

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add liability repo + handler instantiation**

After line `assetHandler := handlers.NewAssetHandler(assetRepo)` (around line 54), add:

```go
liabilityRepo    := repo.NewLiabilityRepo(db)
liabilityHandler := handlers.NewLiabilityHandler(liabilityRepo)
```

- [ ] **Step 2: Register routes**

After the assets routes block (after line 111 `r.Delete("/assets/{id}", assetHandler.Delete)`), add:

```go
r.Get("/liabilities",          liabilityHandler.List)
r.Post("/liabilities",          liabilityHandler.Create)
r.Patch("/liabilities/{id}",    liabilityHandler.Update)
r.Delete("/liabilities/{id}",   liabilityHandler.Delete)
```

- [ ] **Step 3: Build and verify**

```bash
cd backend && go build ./...
```

Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat: register liability routes"
```

---

## Task 7: Frontend — Types + API Endpoints

**Files:**
- Modify: `frontend/src/api/types.ts`
- Modify: `frontend/src/api/endpoints.ts`

- [ ] **Step 1: Add `Liability` type to `types.ts`**

In `frontend/src/api/types.ts`, add after the `Asset` interface:

```ts
export interface Liability {
  id: string
  user_id: string
  name: string
  category: string
  balance: number
  original_principal: number | null
  interest_rate: number | null
  started_at: string | null
  due_at: string | null
  notes: string
}
```

- [ ] **Step 2: Add CRUD functions to `endpoints.ts`**

In `frontend/src/api/endpoints.ts`, add the `Liability` import to the existing import line (add `Liability` to the type list), then append at the end of the file:

```ts
export const getLiabilities = () =>
  apiClient.get<Liability[]>('/liabilities').then(r => r.data)
export const createLiability = (data: Omit<Liability, 'id' | 'user_id'>) =>
  apiClient.post<Liability>('/liabilities', data).then(r => r.data)
export const updateLiability = (id: string, data: Partial<Omit<Liability, 'id' | 'user_id'>>) =>
  apiClient.patch<Liability>(`/liabilities/${id}`, data).then(r => r.data)
export const deleteLiability = (id: string) =>
  apiClient.delete(`/liabilities/${id}`)
```

- [ ] **Step 3: Verify frontend compiles**

```bash
cd frontend && npm run build 2>&1 | tail -10
```

Expected: build succeeds.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/api/types.ts frontend/src/api/endpoints.ts
git commit -m "feat: add Liability type and API endpoints"
```

---

## Task 8: Frontend — LiabilitiesTab + Updated Summary Widget

**Files:**
- Modify: `frontend/src/pages/WealthPage.tsx`

- [ ] **Step 1: Add imports at top of WealthPage.tsx**

Add `getLiabilities, createLiability, updateLiability, deleteLiability` to the existing import from `'../api/endpoints'`:

```ts
import {
  getTransactions, createTransaction, deleteTransaction,
  getBudgets, upsertBudget,
  getAssets, createAsset, updateAsset, deleteAsset,
  getLiabilities, createLiability, updateLiability, deleteLiability,
  getNetWorthSnapshots, addNetWorthSnapshot,
  getBenchmarks, getBankRates, getNews, triggerScrape,
} from '../api/endpoints'
import type { Transaction, Asset, Liability, BankRate, NewsItem } from '../api/types'
```

- [ ] **Step 2: Add `LiabilitiesTab` component**

Add this component after the closing `}` of `AssetsTab` (around line 272, before `const BANK_DISPLAY`):

```tsx
const LIABILITY_CATEGORIES = ['Mortgage', 'Car Loan', 'Credit Card', 'Personal Loan', 'Student Loan', 'Other']

interface LiabilityFormValues {
  name: string
  category: string
  balance: number
  original_principal: number | null
  interest_rate_pct: number | null
  started_at: string | null
  due_at: string | null
  notes: string
}

function buildLiabilityPayload(values: LiabilityFormValues) {
  return {
    name: values.name,
    category: values.category,
    balance: values.balance,
    original_principal: values.original_principal ?? null,
    interest_rate: values.interest_rate_pct != null ? values.interest_rate_pct / 100 : null,
    started_at: values.started_at || null,
    due_at: values.due_at || null,
    notes: values.notes || '',
  }
}

function liabilityFormFields(form: FormInstance<LiabilityFormValues>, onFinish: (v: LiabilityFormValues) => void, loading: boolean) {
  return (
    <Form form={form} layout="vertical" onFinish={onFinish}>
      <Form.Item name="name" label="Name" rules={[{ required: true, message: 'Name is required' }]}><Input /></Form.Item>
      <Form.Item name="category" label="Category" rules={[{ required: true, message: 'Category is required' }]}>
        <Select options={LIABILITY_CATEGORIES.map(c => ({ value: c, label: c }))} />
      </Form.Item>
      <Form.Item name="balance" label="Current Balance (₫)" rules={[{ required: true, message: 'Balance is required' }, { type: 'number', min: 0, message: 'Must be >= 0' }]}>
        <InputNumber min={0} step={1000000} style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item name="original_principal" label="Original Principal (₫)">
        <InputNumber min={0} step={1000000} style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item name="interest_rate_pct" label="Interest Rate (% per year)">
        <InputNumber min={0} max={100} step={0.1} style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item name="started_at" label="Start Date"><Input type="date" /></Form.Item>
      <Form.Item name="due_at" label="Due Date"><Input type="date" /></Form.Item>
      <Form.Item name="notes" label="Notes"><Input.TextArea rows={2} /></Form.Item>
      <Button type="primary" htmlType="submit" loading={loading} block>Save</Button>
    </Form>
  )
}

function LiabilitiesTab() {
  const [addOpen, setAddOpen] = useState(false)
  const [editItem, setEditItem] = useState<Liability | null>(null)
  const [addForm] = Form.useForm<LiabilityFormValues>()
  const [editForm] = Form.useForm<LiabilityFormValues>()
  const qc = useQueryClient()

  const { data: liabilities = [], isLoading } = useQuery({ queryKey: ['liabilities'], queryFn: getLiabilities })

  const addMutation = useMutation({
    mutationFn: (values: LiabilityFormValues) => createLiability(buildLiabilityPayload(values)),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['liabilities'] }); setAddOpen(false); addForm.resetFields() },
  })
  const editMutation = useMutation({
    mutationFn: ({ id, values }: { id: string; values: LiabilityFormValues }) => updateLiability(id, buildLiabilityPayload(values)),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['liabilities'] }); setEditItem(null) },
  })
  const deleteMutation = useMutation({
    mutationFn: deleteLiability,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['liabilities'] }),
  })

  const totalBalance = liabilities.reduce((s, l) => s + l.balance, 0)

  const columns: ColumnsType<Liability> = [
    { title: 'Name', dataIndex: 'name', ellipsis: true },
    { title: 'Category', dataIndex: 'category', width: 130 },
    { title: 'Balance', dataIndex: 'balance', width: 150, align: 'right',
      render: v => <span style={{ color: '#ff4d4f', fontWeight: 600 }}>{fmtVND(v)}</span> },
    { title: 'Interest', dataIndex: 'interest_rate', width: 90, align: 'right',
      render: v => v != null ? `${(v * 100).toFixed(1)}%` : '—' },
    { title: 'Due', dataIndex: 'due_at', width: 110, render: v => v ?? '—' },
    {
      title: '', width: 72,
      render: (_, row) => (
        <>
          <Button type="text" size="small" icon={<EditOutlined />} onClick={() => {
            setEditItem(row)
            editForm.setFieldsValue({
              name: row.name,
              category: row.category,
              balance: row.balance,
              original_principal: row.original_principal,
              interest_rate_pct: row.interest_rate != null ? Math.round(row.interest_rate * 1000) / 10 : null,
              started_at: row.started_at,
              due_at: row.due_at,
              notes: row.notes,
            })
          }} />
          <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(row.id)} />
        </>
      ),
    },
  ]

  return (
    <>
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        <Col span={8}>
          <Card size="small">
            <div style={{ fontSize: 12, color: '#999' }}>Total Liabilities</div>
            <div style={{ fontSize: 22, fontWeight: 700, color: '#ff4d4f' }}>{fmtVND(totalBalance)}</div>
          </Card>
        </Col>
      </Row>
      <Card size="small" title="Liabilities" extra={<Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add</Button>}>
        {isLoading ? <Spin /> : <Table dataSource={liabilities} columns={columns} size="small" rowKey="id" pagination={{ pageSize: 20 }} />}
      </Card>
      <Modal title="Add Liability" open={addOpen} onCancel={() => { setAddOpen(false); addForm.resetFields() }} footer={null}>
        {liabilityFormFields(addForm, values => addMutation.mutate(values), addMutation.isPending)}
      </Modal>
      <Drawer title="Edit Liability" open={editItem !== null} onClose={() => setEditItem(null)} width={400} footer={null}>
        {editItem && liabilityFormFields(editForm, values => editMutation.mutate({ id: editItem.id, values }), editMutation.isPending)}
      </Drawer>
    </>
  )
}
```

- [ ] **Step 3: Update the net worth summary widget and add Liabilities tab**

Find the `AssetsTab` summary widget (the `<Row>` with "Total Assets" card around line 239) and replace it with a three-card summary that includes liabilities and net worth:

```tsx
// In AssetsTab, replace the single "Total Assets" Col:
// Before:
//   <Col span={6}>
//     <Card size="small">
//       <div style={{ fontSize: 12, color: '#999' }}>Total Assets</div>
//       <div style={{ fontSize: 22, fontWeight: 700, color: '#52c41a' }}>{fmtVND(grandTotal)}</div>
//     </Card>
//   </Col>

// After (add liabilities query and update the summary row):
```

Add a liabilities query inside `AssetsTab` just after the assets query:

```tsx
const { data: liabilities = [] } = useQuery({ queryKey: ['liabilities'], queryFn: getLiabilities })
const totalLiabilities = liabilities.reduce((s, l) => s + l.balance, 0)
const netWorth = grandTotal - totalLiabilities
```

Then replace the single "Total Assets" `<Col span={6}>` with:

```tsx
<Col span={6}>
  <Card size="small">
    <div style={{ fontSize: 12, color: '#999' }}>Total Assets</div>
    <div style={{ fontSize: 22, fontWeight: 700, color: '#52c41a' }}>{fmtVND(grandTotal)}</div>
  </Card>
</Col>
<Col span={6}>
  <Card size="small">
    <div style={{ fontSize: 12, color: '#999' }}>Total Liabilities</div>
    <div style={{ fontSize: 22, fontWeight: 700, color: '#ff4d4f' }}>{fmtVND(totalLiabilities)}</div>
  </Card>
</Col>
<Col span={6}>
  <Card size="small">
    <div style={{ fontSize: 12, color: '#999' }}>Net Worth</div>
    <div style={{ fontSize: 22, fontWeight: 700, color: netWorth >= 0 ? '#1677ff' : '#ff4d4f' }}>{fmtVND(netWorth)}</div>
  </Card>
</Col>
```

Note: the existing `{categories.slice(0, 3).map(...)}` category breakdown cols remain; they'll now start at the 4th col position. If `span={6}` totals exceed 24, remove one category breakdown col or reduce to `slice(0, 0)` — the three summary cards already give the key data.

Actually to avoid layout overflow, remove the per-category breakdown cards from the summary row (the `{categories.slice(0, 3).map(...)}` block) entirely, since the three key cards (Assets / Liabilities / Net Worth) replace them:

```tsx
// Remove this block entirely from AssetsTab:
// {categories.slice(0, 3).map(cat => { ... })}
```

- [ ] **Step 4: Register LiabilitiesTab in the Tabs items array**

Find the tabs items array near the bottom of `WealthPage` (around line 479):

```tsx
{ key: 'assets',       label: 'Assets',       children: <AssetsTab /> },
```

Add after it:

```tsx
{ key: 'liabilities',  label: 'Liabilities',  children: <LiabilitiesTab /> },
```

- [ ] **Step 5: Build frontend**

```bash
cd frontend && npm run build 2>&1 | tail -15
```

Expected: build succeeds with no errors.

- [ ] **Step 6: Run lint**

```bash
cd frontend && npm run lint 2>&1 | tail -10
```

Expected: no errors.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/pages/WealthPage.tsx
git commit -m "feat: add LiabilitiesTab and net worth summary widget"
```

---

## Task 9: Final Checks + PR

- [ ] **Step 1: Run backend tests with coverage**

```bash
cd backend && go test ./internal/handlers/... ./internal/middleware/... -coverprofile=coverage.out -covermode=atomic && bash scripts/hooks/pre-commit
```

Expected: all pass, per-file ≥ 80%.

- [ ] **Step 2: Run frontend lint + build**

```bash
cd frontend && npm run lint && npm run build
```

Expected: clean.

- [ ] **Step 3: Create PR**

```bash
git push -u origin fix/scraper-broken-apis
gh pr create --title "feat: add liabilities tracking" --body "$(cat <<'EOF'
## Summary
- New `liabilities` table with CRUD API (balance, interest rate, start/due dates)
- `LiabilitiesTab` in WealthPage — add/edit/delete debts by category
- Assets tab summary updated to show Assets | Liabilities | Net Worth (Assets − Liabilities)
- Two migrations: embedded backend + Supabase

## Test plan
- [ ] Backend unit tests pass ≥ 80% per file
- [ ] Add a liability, verify it appears in Liabilities tab
- [ ] Edit liability balance, verify update persists
- [ ] Delete liability, verify removed
- [ ] Assets tab shows correct Net Worth = assets − liabilities
- [ ] Frontend lint + build clean

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
gh pr merge --auto --squash
```

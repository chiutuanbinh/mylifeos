# Unified Objectives Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Merge Goals + Habits into a single Objectives page where habits become recurring key results under goals, with a daily gate view and a reminder mockup in Settings.

**Architecture:** DB migration converts `habits` → recurring `key_results`, `habit_logs` → `kr_logs`. Backend gains a KRLog handler and route. Frontend replaces GoalsPage + HealthPage with a single ObjectivesPage containing a daily gate section on top and a goals grid below.

**Tech Stack:** Go/chi/pgx (backend), React/Ant Design/React Query (frontend), PostgreSQL migrations (Supabase + embedded).

---

## File Map

| Action | Path |
|--------|------|
| Create | `supabase/migrations/20260614000001_objectives.sql` |
| Create | `backend/internal/migrate/004_objectives.sql` |
| Modify | `backend/internal/models/models.go` |
| Create | `backend/internal/repo/kr_logs.go` |
| Modify | `backend/internal/repo/goals.go` |
| Modify | `backend/internal/handlers/goals.go` |
| Create | `backend/internal/handlers/kr_logs.go` |
| Create | `backend/internal/handlers/kr_logs_test.go` |
| Modify | `backend/internal/handlers/goals_test.go` |
| Delete | `backend/internal/handlers/habits.go` |
| Delete | `backend/internal/handlers/habits_test.go` |
| Delete | `backend/internal/repo/habits.go` |
| Modify | `backend/internal/repo/dashboard.go` |
| Modify | `backend/cmd/server/main.go` |
| Modify | `frontend/src/api/types.ts` |
| Modify | `frontend/src/api/endpoints.ts` |
| Create | `frontend/src/pages/ObjectivesPage.tsx` |
| Delete | `frontend/src/pages/GoalsPage.tsx` |
| Delete | `frontend/src/pages/HealthPage.tsx` |
| Modify | `frontend/src/App.tsx` |
| Modify | `frontend/src/components/AppShell.tsx` |
| Modify | `frontend/src/pages/SettingsPage.tsx` |

---

## Task 1: DB Migration — Objectives

**Files:**
- Create: `supabase/migrations/20260614000001_objectives.sql`
- Create: `backend/internal/migrate/004_objectives.sql`

- [ ] **Step 1: Write the migration SQL**

Create `supabase/migrations/20260614000001_objectives.sql` with this exact content:

```sql
-- Add recurring + reminder_time to key_results
ALTER TABLE key_results
  ADD COLUMN IF NOT EXISTS recurring      BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS reminder_time  TIME    DEFAULT NULL;

-- KR logs table (replaces habit_logs, keyed by kr_id)
CREATE TABLE IF NOT EXISTS kr_logs (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  kr_id       UUID NOT NULL REFERENCES key_results(id) ON DELETE CASCADE,
  user_id     UUID NOT NULL,
  logged_date DATE NOT NULL,
  done        BOOLEAN NOT NULL DEFAULT TRUE,
  UNIQUE(kr_id, logged_date)
);

ALTER TABLE kr_logs ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS kr_logs_user ON kr_logs;
CREATE POLICY kr_logs_user ON kr_logs USING (user_id = auth.uid());

-- Migrate habits → recurring KRs under a "Daily Routines" goal per user
DO $$
DECLARE
  u        RECORD;
  goal_id  UUID;
  h        RECORD;
  new_kr   UUID;
BEGIN
  FOR u IN SELECT DISTINCT user_id FROM habits LOOP
    -- Create (or find existing) "Daily Routines" goal
    INSERT INTO goals (user_id, name, description, color, status)
    VALUES (u.user_id, 'Daily Routines', 'Auto-migrated from habits', '#52c41a', 'active')
    ON CONFLICT DO NOTHING
    RETURNING id INTO goal_id;

    IF goal_id IS NULL THEN
      SELECT id INTO goal_id
      FROM goals
      WHERE user_id = u.user_id AND name = 'Daily Routines'
      LIMIT 1;
    END IF;

    -- Insert each habit as a recurring KR
    FOR h IN SELECT * FROM habits WHERE user_id = u.user_id LOOP
      INSERT INTO key_results (goal_id, user_id, description, done, recurring)
      VALUES (goal_id, u.user_id, h.icon || ' ' || h.name, FALSE, TRUE)
      RETURNING id INTO new_kr;

      -- Copy habit_logs → kr_logs
      INSERT INTO kr_logs (kr_id, user_id, logged_date, done)
      SELECT new_kr, hl.user_id, hl.logged_date, hl.done
      FROM habit_logs hl
      WHERE hl.habit_id = h.id
      ON CONFLICT (kr_id, logged_date) DO NOTHING;
    END LOOP;
  END LOOP;
END $$;

-- Drop old tables
DROP TABLE IF EXISTS habit_logs;
DROP TABLE IF EXISTS habits;
```

Copy the same content to `backend/internal/migrate/004_objectives.sql`.

- [ ] **Step 2: Verify the migration files exist**

```bash
ls supabase/migrations/20260614000001_objectives.sql
ls backend/internal/migrate/004_objectives.sql
```

Expected: both files listed, no error.

- [ ] **Step 3: Commit**

```bash
git add supabase/migrations/20260614000001_objectives.sql backend/internal/migrate/004_objectives.sql
git commit -m "chore: db migration — objectives (habits → recurring KRs)"
```

---

## Task 2: Update Backend Models

**Files:**
- Modify: `backend/internal/models/models.go`

- [ ] **Step 1: Update `KeyResult` struct — add Recurring and ReminderTime**

In `backend/internal/models/models.go`, replace the `KeyResult` struct:

```go
// Before:
type KeyResult struct {
	ID          string `json:"id"`
	GoalID      string `json:"goal_id"`
	UserID      string `json:"user_id"`
	Description string `json:"description"`
	Done        bool   `json:"done"`
}

// After:
type KeyResult struct {
	ID           string  `json:"id"`
	GoalID       string  `json:"goal_id"`
	UserID       string  `json:"user_id"`
	Description  string  `json:"description"`
	Done         bool    `json:"done"`
	Recurring    bool    `json:"recurring"`
	ReminderTime *string `json:"reminder_time,omitempty"`
}
```

- [ ] **Step 2: Add KRLog struct, remove Habit and HabitLog**

In the same file:
1. Delete the `Habit` struct entirely.
2. Delete the `HabitLog` struct entirely.
3. Add `KRLog` after `KeyResult`:

```go
type KRLog struct {
	ID         string `json:"id"`
	KRID       string `json:"kr_id"`
	UserID     string `json:"user_id"`
	LoggedDate string `json:"logged_date"`
	Done       bool   `json:"done"`
}
```

- [ ] **Step 3: Verify build compiles (will fail on habit references — expected)**

```bash
cd backend && go build ./... 2>&1 | head -30
```

Expected: errors about `models.Habit`, `models.HabitLog` undefined. That's correct — we'll fix them in the next tasks.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/models/models.go
git commit -m "refactor: replace Habit/HabitLog models with KRLog, add recurring to KeyResult"
```

---

## Task 3: Create KRLog Repo

**Files:**
- Create: `backend/internal/repo/kr_logs.go`
- Delete: `backend/internal/repo/habits.go`

- [ ] **Step 1: Create `backend/internal/repo/kr_logs.go`**

```go
package repo

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type KRLogRepo interface {
	GetLogs(ctx context.Context, userID, date string) ([]models.KRLog, error)
	GetLogRange(ctx context.Context, krID, userID, from, to string) ([]models.KRLog, error)
	ToggleLog(ctx context.Context, krID, userID, date string) (models.KRLog, error)
}

type pgKRLogRepo struct{ db *pgxpool.Pool }

func NewKRLogRepo(db *pgxpool.Pool) KRLogRepo { return &pgKRLogRepo{db} }

func scanKRLog(row interface{ Scan(...any) error }) (models.KRLog, error) {
	var l models.KRLog
	var loggedDate time.Time
	err := row.Scan(&l.ID, &l.KRID, &l.UserID, &loggedDate, &l.Done)
	l.LoggedDate = loggedDate.Format("2006-01-02")
	return l, err
}

func (r *pgKRLogRepo) GetLogs(ctx context.Context, userID, date string) ([]models.KRLog, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	rows, err := r.db.Query(ctx,
		`SELECT kl.id, kl.kr_id, kl.user_id, kl.logged_date, kl.done
		 FROM kr_logs kl
		 JOIN key_results kr ON kr.id = kl.kr_id
		 WHERE kl.user_id = $1 AND kl.logged_date = $2::date AND kr.recurring = TRUE`,
		userID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.KRLog
	for rows.Next() {
		l, err := scanKRLog(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	if out == nil {
		out = []models.KRLog{}
	}
	return out, rows.Err()
}

func (r *pgKRLogRepo) GetLogRange(ctx context.Context, krID, userID, from, to string) ([]models.KRLog, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, kr_id, user_id, logged_date, done
		 FROM kr_logs
		 WHERE kr_id = $1 AND user_id = $2 AND logged_date BETWEEN $3::date AND $4::date
		 ORDER BY logged_date`,
		krID, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.KRLog
	for rows.Next() {
		l, err := scanKRLog(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	if out == nil {
		out = []models.KRLog{}
	}
	return out, rows.Err()
}

func (r *pgKRLogRepo) ToggleLog(ctx context.Context, krID, userID, date string) (models.KRLog, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	row := r.db.QueryRow(ctx,
		`INSERT INTO kr_logs (kr_id, user_id, logged_date, done)
		 VALUES ($1, $2, $3::date, TRUE)
		 ON CONFLICT (kr_id, logged_date)
		 DO UPDATE SET done = NOT kr_logs.done
		 RETURNING id, kr_id, user_id, logged_date, done`,
		krID, userID, date)
	return scanKRLog(row)
}
```

- [ ] **Step 2: Delete `backend/internal/repo/habits.go`**

```bash
rm backend/internal/repo/habits.go
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/repo/kr_logs.go backend/internal/repo/habits.go
git commit -m "refactor: replace HabitRepo with KRLogRepo"
```

---

## Task 4: Update Goals Repo — Scan Recurring Fields

**Files:**
- Modify: `backend/internal/repo/goals.go`

- [ ] **Step 1: Update `computeProgress` to exclude recurring KRs**

In `backend/internal/repo/goals.go`, replace `computeProgress`:

```go
func computeProgress(krs []models.KeyResult) int {
	// Only one-time KRs count toward goal progress
	oneTime := make([]models.KeyResult, 0, len(krs))
	for _, kr := range krs {
		if !kr.Recurring {
			oneTime = append(oneTime, kr)
		}
	}
	if len(oneTime) == 0 {
		return 0
	}
	done := 0
	for _, kr := range oneTime {
		if kr.Done {
			done++
		}
	}
	return int(float64(done) / float64(len(oneTime)) * 100)
}
```

- [ ] **Step 2: Update KR scan query and scan call in `List`**

In the `List` method, replace the KR query and scan:

```go
// Replace the krows query:
krows, err := r.db.Query(ctx,
    `SELECT id, goal_id, user_id, description, done, recurring,
            TO_CHAR(reminder_time, 'HH24:MI') AS reminder_time
     FROM key_results WHERE goal_id = $1 ORDER BY created_at`, g.ID)

// Replace the scan inside the loop:
for krows.Next() {
    var kr models.KeyResult
    krows.Scan(&kr.ID, &kr.GoalID, &kr.UserID, &kr.Description, &kr.Done, &kr.Recurring, &kr.ReminderTime)
    krs = append(krs, kr)
}
```

- [ ] **Step 3: Update `AddKeyResult` to insert recurring + reminder_time**

Replace the `AddKeyResult` method:

```go
func (r *pgGoalRepo) AddKeyResult(ctx context.Context, kr models.KeyResult) (models.KeyResult, error) {
	var reminderExpr string
	if kr.ReminderTime != nil && *kr.ReminderTime != "" {
		reminderExpr = *kr.ReminderTime
	}
	var reminderArg interface{} = nil
	if reminderExpr != "" {
		reminderArg = reminderExpr
	}
	row := r.db.QueryRow(ctx,
		`INSERT INTO key_results (goal_id, user_id, description, done, recurring, reminder_time)
		 VALUES ($1, $2, $3, FALSE, $4, $5::time)
		 RETURNING id, goal_id, user_id, description, done, recurring,
		           TO_CHAR(reminder_time, 'HH24:MI')`,
		kr.GoalID, kr.UserID, kr.Description, kr.Recurring, reminderArg)
	var out models.KeyResult
	err := row.Scan(&out.ID, &out.GoalID, &out.UserID, &out.Description, &out.Done, &out.Recurring, &out.ReminderTime)
	return out, err
}
```

- [ ] **Step 4: Update `UpdateKeyResult` to persist recurring + reminder_time**

Replace `UpdateKeyResult`:

```go
func (r *pgGoalRepo) UpdateKeyResult(ctx context.Context, kr models.KeyResult) (models.KeyResult, error) {
	var reminderArg interface{} = nil
	if kr.ReminderTime != nil && *kr.ReminderTime != "" {
		reminderArg = *kr.ReminderTime
	}
	row := r.db.QueryRow(ctx,
		`UPDATE key_results
		 SET description=$1, done=$2, recurring=$3, reminder_time=$4::time
		 WHERE id=$5 AND user_id=$6
		 RETURNING id, goal_id, user_id, description, done, recurring,
		           TO_CHAR(reminder_time, 'HH24:MI')`,
		kr.Description, kr.Done, kr.Recurring, reminderArg, kr.ID, kr.UserID)
	var out models.KeyResult
	err := row.Scan(&out.ID, &out.GoalID, &out.UserID, &out.Description, &out.Done, &out.Recurring, &out.ReminderTime)
	return out, err
}
```

- [ ] **Step 5: Verify no compile errors in repo package**

```bash
cd backend && go build ./internal/repo/... 2>&1
```

Expected: no output (clean build).

- [ ] **Step 6: Commit**

```bash
git add backend/internal/repo/goals.go
git commit -m "refactor: goals repo scans recurring/reminder_time, progress excludes recurring KRs"
```

---

## Task 5: Update Dashboard Repo — Use KR Logs

**Files:**
- Modify: `backend/internal/repo/dashboard.go`

- [ ] **Step 1: Replace habit_logs queries with kr_logs**

In `backend/internal/repo/dashboard.go`, replace the habits count query:

```go
// Replace:
row := r.db.QueryRow(ctx,
    `SELECT COUNT(*), COALESCE(SUM(CASE WHEN hl.done THEN 1 ELSE 0 END), 0)
     FROM habits h
     LEFT JOIN habit_logs hl ON hl.habit_id = h.id AND hl.logged_date = CURRENT_DATE
     WHERE h.user_id = $1`, userID)
row.Scan(&s.HabitsTotal, &s.HabitsDoneToday)

// With:
row := r.db.QueryRow(ctx,
    `SELECT COUNT(*), COALESCE(SUM(CASE WHEN kl.done THEN 1 ELSE 0 END), 0)
     FROM key_results kr
     LEFT JOIN kr_logs kl ON kl.kr_id = kr.id AND kl.logged_date = CURRENT_DATE
     WHERE kr.user_id = $1 AND kr.recurring = TRUE`, userID)
row.Scan(&s.HabitsTotal, &s.HabitsDoneToday)
```

- [ ] **Step 2: Build and verify**

```bash
cd backend && go build ./... 2>&1
```

Expected: remaining errors only in handlers (habits.go still exists). That's expected — next tasks fix it.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/repo/dashboard.go
git commit -m "refactor: dashboard repo uses kr_logs instead of habit_logs"
```

---

## Task 6: Create KRLog Handler + Delete Habits Handler

**Files:**
- Create: `backend/internal/handlers/kr_logs.go`
- Delete: `backend/internal/handlers/habits.go`

- [ ] **Step 1: Create `backend/internal/handlers/kr_logs.go`**

```go
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/repo"
)

type KRLogHandler struct{ repo repo.KRLogRepo }

func NewKRLogHandler(r repo.KRLogRepo) *KRLogHandler { return &KRLogHandler{r} }

func (h *KRLogHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	date := r.URL.Query().Get("date")
	logs, err := h.repo.GetLogs(r.Context(), uid, date)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (h *KRLogHandler) GetLogRange(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		http.Error(w, `{"error":"from and to are required"}`, 400)
		return
	}
	logs, err := h.repo.GetLogRange(r.Context(), chi.URLParam(r, "id"), uid, from, to)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (h *KRLogHandler) ToggleLog(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var body struct {
		Date string `json:"date"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	log, err := h.repo.ToggleLog(r.Context(), chi.URLParam(r, "id"), uid, body.Date)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(log)
}
```

- [ ] **Step 2: Delete habits handler**

```bash
rm backend/internal/handlers/habits.go
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/handlers/kr_logs.go backend/internal/handlers/habits.go
git commit -m "refactor: replace HabitHandler with KRLogHandler"
```

---

## Task 7: Update Goals Handler — Accept Recurring Fields

**Files:**
- Modify: `backend/internal/handlers/goals.go`

- [ ] **Step 1: Read the current `AddKeyResult` handler**

Open `backend/internal/handlers/goals.go` and find `AddKeyResult`. It currently decodes only `description`. Update the body struct and model construction:

```go
func (h *GoalHandler) AddKeyResult(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	goalID := chi.URLParam(r, "id")
	var body struct {
		Description  string  `json:"description"`
		Recurring    bool    `json:"recurring"`
		ReminderTime *string `json:"reminder_time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Description == "" {
		http.Error(w, `{"error":"description required"}`, 400)
		return
	}
	kr, err := h.repo.AddKeyResult(r.Context(), models.KeyResult{
		GoalID:       goalID,
		UserID:       uid,
		Description:  body.Description,
		Recurring:    body.Recurring,
		ReminderTime: body.ReminderTime,
	})
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(kr)
}
```

- [ ] **Step 2: Update `UpdateKeyResult` handler body struct**

Find `UpdateKeyResult` in `goals.go` and update the body struct to include recurring and reminder_time:

```go
func (h *GoalHandler) UpdateKeyResult(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var body struct {
		Description  string  `json:"description"`
		Done         bool    `json:"done"`
		Recurring    bool    `json:"recurring"`
		ReminderTime *string `json:"reminder_time"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	kr, err := h.repo.UpdateKeyResult(r.Context(), models.KeyResult{
		ID:           chi.URLParam(r, "kr_id"),
		UserID:       uid,
		Description:  body.Description,
		Done:         body.Done,
		Recurring:    body.Recurring,
		ReminderTime: body.ReminderTime,
	})
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(kr)
}
```

- [ ] **Step 3: Build the full backend — should be clean now**

```bash
cd backend && go build ./... 2>&1
```

Expected: no output (clean build).

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handlers/goals.go
git commit -m "feat: goals handler accepts recurring + reminder_time on key results"
```

---

## Task 8: Write KRLog Handler Tests + Update Goals Tests

**Files:**
- Create: `backend/internal/handlers/kr_logs_test.go`
- Delete: `backend/internal/handlers/habits_test.go`
- Modify: `backend/internal/handlers/goals_test.go`

- [ ] **Step 1: Delete old habits test file**

```bash
rm backend/internal/handlers/habits_test.go
```

- [ ] **Step 2: Create `backend/internal/handlers/kr_logs_test.go`**

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

type mockKRLogRepo struct{}

func (m *mockKRLogRepo) GetLogs(_ context.Context, _, _ string) ([]models.KRLog, error) {
	return []models.KRLog{{ID: "kl-1", KRID: "kr-1", Done: true, LoggedDate: "2026-06-14"}}, nil
}
func (m *mockKRLogRepo) GetLogRange(_ context.Context, _, _, _, _ string) ([]models.KRLog, error) {
	return []models.KRLog{}, nil
}
func (m *mockKRLogRepo) ToggleLog(_ context.Context, _, _, _ string) (models.KRLog, error) {
	return models.KRLog{ID: "kl-1", KRID: "kr-1", Done: true, LoggedDate: "2026-06-14"}, nil
}

func TestKRLogGetLogs(t *testing.T) {
	devEnv(t)
	h := handlers.NewKRLogHandler(&mockKRLogRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.GetLogs))

	req := httptest.NewRequest("GET", "/api/v1/kr-logs?date=2026-06-14", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var logs []models.KRLog
	if err := json.NewDecoder(w.Body).Decode(&logs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(logs) != 1 || logs[0].KRID != "kr-1" {
		t.Fatalf("unexpected: %+v", logs)
	}
}

func TestKRLogToggle(t *testing.T) {
	devEnv(t)
	h := handlers.NewKRLogHandler(&mockKRLogRepo{})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "kr-1")

	body, _ := json.Marshal(map[string]string{"date": "2026-06-14"})
	req := httptest.NewRequest("POST", "/api/v1/key-results/kr-1/log", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := middleware.Auth(http.HandlerFunc(h.ToggleLog))
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var log models.KRLog
	if err := json.NewDecoder(w.Body).Decode(&log); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !log.Done {
		t.Fatalf("expected done=true")
	}
}

func TestKRLogGetLogRange(t *testing.T) {
	devEnv(t)
	h := handlers.NewKRLogHandler(&mockKRLogRepo{})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "kr-1")

	req := httptest.NewRequest("GET", "/api/v1/key-results/kr-1/logs?from=2026-06-01&to=2026-06-14", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler := middleware.Auth(http.HandlerFunc(h.GetLogRange))
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestKRLogGetLogRange_MissingParams(t *testing.T) {
	devEnv(t)
	h := handlers.NewKRLogHandler(&mockKRLogRepo{})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "kr-1")

	req := httptest.NewRequest("GET", "/api/v1/key-results/kr-1/logs", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler := middleware.Auth(http.HandlerFunc(h.GetLogRange))
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
```

- [ ] **Step 3: Update goals_test.go mock to include recurring fields**

In `backend/internal/handlers/goals_test.go`, find the `mockGoalRepo.AddKeyResult` mock method and update it to handle `Recurring` and `ReminderTime`:

```go
func (m *mockGoalRepo) AddKeyResult(_ context.Context, kr models.KeyResult) (models.KeyResult, error) {
	kr.ID = "kr-new"
	return kr, nil
}
func (m *mockGoalRepo) UpdateKeyResult(_ context.Context, kr models.KeyResult) (models.KeyResult, error) {
	return kr, nil
}
```

Also update the mock `List` return to include `Recurring` field in any `KeyResult` values:

```go
func (m *mockGoalRepo) List(_ context.Context, _ string) ([]models.Goal, error) {
	return []models.Goal{{
		ID: "g-1", Name: "Test Goal", Status: "active", Color: "#1677ff",
		KeyResults: []models.KeyResult{
			{ID: "kr-1", Description: "Write tests", Done: false, Recurring: false},
		},
	}}, nil
}
```

- [ ] **Step 4: Run tests**

```bash
cd backend && go test ./internal/handlers/... -v 2>&1 | tail -30
```

Expected: all tests PASS.

- [ ] **Step 5: Run coverage check**

```bash
cd backend && go test ./internal/handlers/... ./internal/middleware/... -coverprofile=coverage.out -covermode=atomic && bash scripts/hooks/pre-commit
```

Expected: `✓ Coverage OK`, all files ≥80%.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/handlers/kr_logs_test.go backend/internal/handlers/habits_test.go backend/internal/handlers/goals_test.go
git commit -m "test: kr_logs handler tests, update goals mock for recurring fields"
```

---

## Task 9: Update Router

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Replace habit handler with KRLog handler in main.go**

In `backend/cmd/server/main.go`:

1. Remove the `habitHandler` line and all `/habits` routes.
2. Add `krLogHandler` and new routes.

Replace the handler setup section:

```go
// Remove:
habitHandler   := handlers.NewHabitHandler(repo.NewHabitRepo(db))

// Add:
krLogHandler   := handlers.NewKRLogHandler(repo.NewKRLogRepo(db))
```

Replace the habits route block:

```go
// Remove:
r.Get("/habits",           habitHandler.List)
r.Post("/habits",           habitHandler.Create)
r.Put("/habits/{id}",      habitHandler.Update)
r.Delete("/habits/{id}",   habitHandler.Delete)
r.Get("/habits/logs",      habitHandler.GetLogs)
r.Get("/habits/{id}/logs", habitHandler.GetLogRange)
r.Post("/habits/{id}/log", habitHandler.ToggleLog)

// Add:
r.Get("/kr-logs",                      krLogHandler.GetLogs)
r.Get("/key-results/{id}/logs",        krLogHandler.GetLogRange)
r.Post("/key-results/{id}/log",        krLogHandler.ToggleLog)
```

- [ ] **Step 2: Build final check**

```bash
cd backend && go build ./... 2>&1
```

Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat: wire KRLog routes, remove habit routes"
```

---

## Task 10: Update Frontend Types + Endpoints

**Files:**
- Modify: `frontend/src/api/types.ts`
- Modify: `frontend/src/api/endpoints.ts`

- [ ] **Step 1: Update `types.ts` — add KRLog, update KeyResult, remove Habit/HabitLog**

In `frontend/src/api/types.ts`:

1. Delete the `Habit` interface.
2. Delete the `HabitLog` interface.
3. Update `KeyResult`:

```typescript
export interface KeyResult {
  id: string
  goal_id: string
  user_id: string
  description: string
  done: boolean
  recurring: boolean
  reminder_time?: string | null
}
```

4. Add `KRLog` interface after `KeyResult`:

```typescript
export interface KRLog {
  id: string
  kr_id: string
  user_id: string
  logged_date: string
  done: boolean
}
```

5. Update `DashboardSummary` — field names stay the same (`habits_total`, `habits_done_today`), no change needed.

- [ ] **Step 2: Update `endpoints.ts` — replace habit endpoints with kr-log endpoints**

In `frontend/src/api/endpoints.ts`:

1. Remove the import of `Habit`, `HabitLog` from `./types`.
2. Add import of `KRLog`.
3. Delete all `getHabits`, `createHabit`, `deleteHabit`, `updateHabit`, `getHabitLogRange`, `getHabitLogs`, `toggleHabitLog` functions.
4. Add:

```typescript
export const getKRLogs = (date?: string) =>
  apiClient.get<KRLog[]>('/kr-logs', { params: { date } }).then(r => r.data)

export const getKRLogRange = (krId: string, from: string, to: string) =>
  apiClient.get<KRLog[]>(`/key-results/${krId}/logs`, { params: { from, to } }).then(r => r.data)

export const toggleKRLog = (krId: string, date?: string) =>
  apiClient.post<KRLog>(`/key-results/${krId}/log`, { date }).then(r => r.data)
```

5. Update `addKeyResult` to accept `recurring` and `reminder_time`:

```typescript
export const addKeyResult = (goalId: string, description: string, recurring = false, reminderTime?: string) =>
  apiClient.post<KeyResult>(`/goals/${goalId}/key-results`, {
    description,
    recurring,
    reminder_time: reminderTime ?? null,
  }).then(r => r.data)
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api/types.ts frontend/src/api/endpoints.ts
git commit -m "refactor: replace Habit/HabitLog types with KRLog, update KR endpoints"
```

---

## Task 11: Create ObjectivesPage

**Files:**
- Create: `frontend/src/pages/ObjectivesPage.tsx`
- Delete: `frontend/src/pages/GoalsPage.tsx`
- Delete: `frontend/src/pages/HealthPage.tsx`

- [ ] **Step 1: Delete old pages**

```bash
rm frontend/src/pages/GoalsPage.tsx frontend/src/pages/HealthPage.tsx
```

- [ ] **Step 2: Create `frontend/src/pages/ObjectivesPage.tsx`**

```typescript
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Row, Col, Card, Progress, Button, Modal, Form, Input, Select,
  Tag, Spin, Tooltip, Space, Switch, TimePicker,
} from 'antd'
import { PlusOutlined, DeleteOutlined, EditOutlined, FireOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import type { Goal, KRLog, KeyResult } from '../api/types'
import {
  getGoals, createGoal, updateGoal, deleteGoal,
  addKeyResult, updateKeyResult, deleteKeyResult,
  getKRLogs, toggleKRLog, getKRLogRange,
} from '../api/endpoints'

const today = new Date().toISOString().split('T')[0]

const STATUS_COLORS: Record<string, string> = {
  active: 'blue', completed: 'green', archived: 'default',
}

function computeStreak(logs: KRLog[], krId: string): number {
  const doneSet = new Set(logs.filter(l => l.done && l.kr_id === krId).map(l => l.logged_date))
  let streak = 0
  const cursor = new Date(today)
  while (true) {
    const d = cursor.toISOString().split('T')[0]
    if (!doneSet.has(d)) break
    streak++
    cursor.setDate(cursor.getDate() - 1)
  }
  return streak
}

function getMonthDays(year: number, month: number): string[] {
  const days: string[] = []
  const count = new Date(year, month + 1, 0).getDate()
  for (let d = 1; d <= count; d++) {
    days.push(`${year}-${String(month + 1).padStart(2, '0')}-${String(d).padStart(2, '0')}`)
  }
  return days
}

function HeatmapMini({ krId, logs }: { krId: string; logs: KRLog[] }) {
  const now = new Date()
  const days = getMonthDays(now.getFullYear(), now.getMonth())
  const doneSet = new Set(logs.filter(l => l.done && l.kr_id === krId).map(l => l.logged_date))
  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(7, 1fr)', gap: 2, marginTop: 6 }}>
      {Array.from({ length: new Date(now.getFullYear(), now.getMonth(), 1).getDay() }).map((_, i) => (
        <div key={`e${i}`} />
      ))}
      {days.map(date => (
        <Tooltip key={date} title={date}>
          <div style={{
            height: 10, borderRadius: 2,
            background: doneSet.has(date) ? '#52c41a' : '#f0f0f0',
            border: date === today ? '1px solid #1677ff' : 'none',
          }} />
        </Tooltip>
      ))}
    </div>
  )
}

export function ObjectivesPage() {
  const [addGoalOpen, setAddGoalOpen] = useState(false)
  const [editGoal, setEditGoal] = useState<Goal | null>(null)
  const [expandedGoal, setExpandedGoal] = useState<string | null>(null)
  const [newKr, setNewKr] = useState<Record<string, string>>({})
  const [newKrRecurring, setNewKrRecurring] = useState<Record<string, boolean>>({})
  const [addGoalForm] = Form.useForm()
  const [editGoalForm] = Form.useForm()
  const qc = useQueryClient()

  const { data: goals = [], isLoading } = useQuery({ queryKey: ['goals'], queryFn: getGoals })
  const { data: todayLogs = [] } = useQuery({
    queryKey: ['kr-logs', today],
    queryFn: () => getKRLogs(today),
  })

  // All recurring KRs across all goals
  const allRecurring = goals.flatMap(g =>
    (g.key_results ?? []).filter(kr => kr.recurring).map(kr => ({ ...kr, goalName: g.name, goalColor: g.color }))
  )
  const todayDoneSet = new Set(todayLogs.filter(l => l.done).map(l => l.kr_id))
  const totalToday = allRecurring.length
  const doneToday = allRecurring.filter(kr => todayDoneSet.has(kr.id)).length
  const todayPct = totalToday ? Math.round(doneToday / totalToday * 100) : 0

  // Month logs for heatmaps (all KRs)
  const now = new Date()
  const monthFrom = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-01`
  const monthTo = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate()}`
  const allKrIds = goals.flatMap(g => (g.key_results ?? []).filter(kr => kr.recurring).map(kr => kr.id))

  const { data: monthLogs = [] } = useQuery({
    queryKey: ['kr-logs-month', monthFrom, monthTo, allKrIds.join(',')],
    queryFn: async () => {
      const results = await Promise.all(
        allKrIds.map(id => getKRLogRange(id, monthFrom, monthTo))
      )
      return results.flat()
    },
    enabled: allKrIds.length > 0,
  })

  const toggleMutation = useMutation({
    mutationFn: (krId: string) => toggleKRLog(krId, today),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['kr-logs', today] })
      qc.invalidateQueries({ queryKey: ['kr-logs-month', monthFrom, monthTo, allKrIds.join(',')] })
    },
  })

  const createGoalMutation = useMutation({
    mutationFn: createGoal,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['goals'] }); setAddGoalOpen(false); addGoalForm.resetFields() },
  })

  const updateGoalMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Goal> }) => updateGoal(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['goals'] }); setEditGoal(null) },
  })

  const deleteGoalMutation = useMutation({
    mutationFn: deleteGoal,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['goals'] }),
  })

  const toggleKRMutation = useMutation({
    mutationFn: ({ goalId, krId, done }: { goalId: string; krId: string; done: boolean }) =>
      updateKeyResult(goalId, krId, { done }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['goals'] }),
  })

  const addKrMutation = useMutation({
    mutationFn: ({ goalId, description, recurring }: { goalId: string; description: string; recurring: boolean }) =>
      addKeyResult(goalId, description, recurring),
    onSuccess: (_, vars) => {
      qc.invalidateQueries({ queryKey: ['goals'] })
      setNewKr(prev => ({ ...prev, [vars.goalId]: '' }))
      setNewKrRecurring(prev => ({ ...prev, [vars.goalId]: false }))
    },
  })

  const deleteKrMutation = useMutation({
    mutationFn: ({ goalId, krId }: { goalId: string; krId: string }) => deleteKeyResult(goalId, krId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['goals'] }),
  })

  const openEditGoal = (g: Goal) => {
    setEditGoal(g)
    editGoalForm.setFieldsValue({
      name: g.name, description: g.description, color: g.color,
      status: g.status, target_date: g.target_date ?? '',
    })
  }

  // Group recurring KRs by goal for the gate section
  const gateGroups = goals
    .map(g => ({
      goal: g,
      krs: (g.key_results ?? []).filter(kr => kr.recurring),
    }))
    .filter(grp => grp.krs.length > 0)

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />

  return (
    <div>
      {/* ── TODAY GATE ── */}
      {totalToday > 0 && (
        <Card
          size="small"
          style={{ marginBottom: 16, borderLeft: '3px solid #1677ff' }}
          title={
            <Space>
              <span style={{ fontWeight: 600 }}>Today</span>
              <span style={{ color: '#999', fontSize: 12 }}>{doneToday}/{totalToday} done</span>
            </Space>
          }
        >
          <Progress percent={todayPct} size="small" strokeColor="#1677ff" style={{ marginBottom: 12 }} />
          {gateGroups.map(({ goal, krs }) => (
            <div key={goal.id} style={{ marginBottom: 12 }}>
              <div style={{ fontSize: 11, color: goal.color, fontWeight: 600, marginBottom: 6, textTransform: 'uppercase', letterSpacing: 0.5 }}>
                {goal.name}
              </div>
              {krs.map(kr => {
                const done = todayDoneSet.has(kr.id)
                const streak = computeStreak([...todayLogs, ...monthLogs], kr.id)
                return (
                  <div key={kr.id} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '6px 0', borderBottom: '1px solid #f5f5f5' }}>
                    <div
                      onClick={() => toggleMutation.mutate(kr.id)}
                      style={{
                        width: 20, height: 20, borderRadius: '50%', cursor: 'pointer', flexShrink: 0,
                        background: done ? '#52c41a' : '#f0f0f0',
                        border: done ? 'none' : '1.5px solid #d9d9d9',
                        display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 10, color: '#fff',
                      }}
                    >{done ? '✓' : ''}</div>
                    <span style={{ fontSize: 13, flex: 1, textDecoration: done ? 'line-through' : 'none', color: done ? '#bbb' : '#222' }}>
                      {kr.description}
                    </span>
                    {streak > 0 && (
                      <Tag color="orange" icon={<FireOutlined />} style={{ fontSize: 11, margin: 0 }}>{streak}d</Tag>
                    )}
                  </div>
                )
              })}
            </div>
          ))}
        </Card>
      )}

      {/* ── GOALS GRID ── */}
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddGoalOpen(true)}>Add Goal</Button>
      </div>

      <Row gutter={[12, 12]}>
        {goals.map(g => {
          const oneTimeKRs = (g.key_results ?? []).filter(kr => !kr.recurring)
          const recurringKRs = (g.key_results ?? []).filter(kr => kr.recurring)
          const recurringDone = recurringKRs.filter(kr => todayDoneSet.has(kr.id)).length
          const expanded = expandedGoal === g.id
          return (
            <Col span={8} key={g.id}>
              <Card
                size="small"
                style={{ borderTop: `3px solid ${g.color}` }}
                title={
                  <Space size={6}>
                    <span style={{ fontSize: 13, fontWeight: 600 }}>{g.name}</span>
                    <Tag color={STATUS_COLORS[g.status]} style={{ fontSize: 11, margin: 0 }}>{g.status}</Tag>
                  </Space>
                }
                extra={
                  <Space size={2}>
                    <Button type="text" size="small" icon={<EditOutlined />} onClick={() => openEditGoal(g)} />
                    <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteGoalMutation.mutate(g.id)} />
                  </Space>
                }
              >
                <Progress percent={g.progress} strokeColor={g.color} size="small" style={{ marginBottom: 8 }} />
                <div style={{ fontSize: 11, color: '#999', marginBottom: 6 }}>
                  {oneTimeKRs.filter(kr => kr.done).length}/{oneTimeKRs.length} key results · {g.progress}%
                </div>

                {/* One-time KRs */}
                {oneTimeKRs.map(kr => (
                  <div key={kr.id} style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                    <div
                      onClick={() => toggleKRMutation.mutate({ goalId: g.id, krId: kr.id, done: !kr.done })}
                      style={{
                        width: 16, height: 16, borderRadius: 3, cursor: 'pointer', flexShrink: 0,
                        background: kr.done ? g.color : '#f0f0f0',
                        border: kr.done ? 'none' : '1.5px solid #d9d9d9',
                        display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 10, color: '#fff',
                      }}
                    >{kr.done ? '✓' : ''}</div>
                    <span style={{ fontSize: 12, flex: 1, textDecoration: kr.done ? 'line-through' : 'none', color: kr.done ? '#bbb' : '#222' }}>
                      {kr.description}
                    </span>
                    <Button type="text" size="small" danger icon={<DeleteOutlined />} style={{ padding: 0, height: 16, width: 16 }}
                      onClick={() => deleteKrMutation.mutate({ goalId: g.id, krId: kr.id })} />
                  </div>
                ))}

                {/* Recurring KRs summary */}
                {recurringKRs.length > 0 && (
                  <div
                    style={{ marginTop: 8, cursor: 'pointer', padding: '4px 0', borderTop: '1px solid #f5f5f5' }}
                    onClick={() => setExpandedGoal(expanded ? null : g.id)}
                  >
                    <span style={{ fontSize: 11, color: '#1677ff' }}>
                      Daily {recurringDone}/{recurringKRs.length} done today {expanded ? '▲' : '▼'}
                    </span>
                    {expanded && (
                      <div style={{ marginTop: 8 }}>
                        {recurringKRs.map(kr => (
                          <div key={kr.id} style={{ marginBottom: 8 }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 2 }}>
                              <span style={{ fontSize: 12, flex: 1 }}>{kr.description}</span>
                              <Button type="text" size="small" danger icon={<DeleteOutlined />} style={{ padding: 0, height: 16, width: 16 }}
                                onClick={e => { e.stopPropagation(); deleteKrMutation.mutate({ goalId: g.id, krId: kr.id }) }} />
                            </div>
                            <HeatmapMini krId={kr.id} logs={monthLogs} />
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}

                {/* Add KR row */}
                <div style={{ display: 'flex', gap: 4, marginTop: 8, alignItems: 'center' }}>
                  <Switch
                    size="small"
                    checked={newKrRecurring[g.id] ?? false}
                    onChange={v => setNewKrRecurring(prev => ({ ...prev, [g.id]: v }))}
                    checkedChildren="🔁" unCheckedChildren="1×"
                  />
                  <Input
                    size="small"
                    placeholder={newKrRecurring[g.id] ? 'Daily habit…' : 'Add key result…'}
                    value={newKr[g.id] ?? ''}
                    onChange={e => setNewKr(prev => ({ ...prev, [g.id]: e.target.value }))}
                    onPressEnter={() => {
                      const desc = (newKr[g.id] ?? '').trim()
                      if (desc) addKrMutation.mutate({ goalId: g.id, description: desc, recurring: newKrRecurring[g.id] ?? false })
                    }}
                    style={{ fontSize: 12 }}
                  />
                  <Button size="small" icon={<PlusOutlined />} onClick={() => {
                    const desc = (newKr[g.id] ?? '').trim()
                    if (desc) addKrMutation.mutate({ goalId: g.id, description: desc, recurring: newKrRecurring[g.id] ?? false })
                  }} />
                </div>
              </Card>
            </Col>
          )
        })}
        {goals.length === 0 && (
          <Col span={24}><div style={{ color: '#bbb', textAlign: 'center', padding: 40 }}>No goals yet. Add your first!</div></Col>
        )}
      </Row>

      {/* Add Goal Modal */}
      <Modal title="Add Goal" open={addGoalOpen} onCancel={() => setAddGoalOpen(false)} footer={null}>
        <Form form={addGoalForm} layout="vertical"
          onFinish={values => createGoalMutation.mutate({ ...values, key_results: [], status: 'active', progress: 0 })}>
          <Form.Item name="name" label="Goal name" rules={[{ required: true, max: 100 }]}><Input /></Form.Item>
          <Form.Item name="description" label="Description"><Input.TextArea rows={2} /></Form.Item>
          <Form.Item name="target_date" label="Target date"><Input type="date" /></Form.Item>
          <Form.Item name="color" label="Color" initialValue="#1677ff"><Input type="color" /></Form.Item>
          <Button type="primary" htmlType="submit" loading={createGoalMutation.isPending} block>Save</Button>
        </Form>
      </Modal>

      {/* Edit Goal Modal */}
      <Modal title="Edit Goal" open={!!editGoal} onCancel={() => setEditGoal(null)} footer={null}>
        <Form form={editGoalForm} layout="vertical"
          onFinish={values => editGoal && updateGoalMutation.mutate({ id: editGoal.id, data: values })}>
          <Form.Item name="name" label="Goal name" rules={[{ required: true, max: 100 }]}><Input /></Form.Item>
          <Form.Item name="description" label="Description"><Input.TextArea rows={2} /></Form.Item>
          <Form.Item name="target_date" label="Target date"><Input type="date" /></Form.Item>
          <Form.Item name="status" label="Status">
            <Select options={[
              { value: 'active', label: 'Active' },
              { value: 'completed', label: 'Completed' },
              { value: 'archived', label: 'Archived' },
            ]} />
          </Form.Item>
          <Form.Item name="color" label="Color"><Input type="color" /></Form.Item>
          <Button type="primary" htmlType="submit" loading={updateGoalMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/ObjectivesPage.tsx frontend/src/pages/GoalsPage.tsx frontend/src/pages/HealthPage.tsx
git commit -m "feat: ObjectivesPage — unified goals + recurring KRs with daily gate"
```

---

## Task 12: Update App + AppShell Navigation

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/AppShell.tsx`

- [ ] **Step 1: Update `App.tsx`**

Replace imports and routes:

```typescript
// Remove:
import { HealthPage } from './pages/HealthPage'
import { GoalsPage } from './pages/GoalsPage'

// Add:
import { ObjectivesPage } from './pages/ObjectivesPage'
```

In the Routes section, replace:
```typescript
// Remove:
<Route path="/health"    element={<HealthPage />} />
<Route path="/goals"     element={<GoalsPage />} />

// Add:
<Route path="/objectives" element={<ObjectivesPage />} />
```

- [ ] **Step 2: Update `AppShell.tsx` NAV array and TITLES**

In `frontend/src/components/AppShell.tsx`, replace the Health and Goals entries:

```typescript
// In NAV array, replace:
{ key: '/health',   icon: <HeartOutlined />,    label: 'Health & Habits' },
{ key: '/goals',    icon: <TrophyOutlined />,   label: 'Goals & OKRs' },

// With:
{ key: '/objectives', icon: <TrophyOutlined />, label: 'Objectives' },
```

In TITLES, replace:
```typescript
// Remove:
'/health': 'Health & Habits',
'/goals': 'Goals & OKRs',

// Add:
'/objectives': 'Objectives',
```

Also remove the `HeartOutlined` import from antd icons (if no longer used elsewhere).

- [ ] **Step 3: Run lint + build**

```bash
cd frontend && npm run lint && npm run build 2>&1 | tail -20
```

Expected: clean (no errors).

- [ ] **Step 4: Commit**

```bash
git add frontend/src/App.tsx frontend/src/components/AppShell.tsx
git commit -m "feat: route /objectives, remove /health and /goals routes"
```

---

## Task 13: Reminders Mockup in Settings

**Files:**
- Modify: `frontend/src/pages/SettingsPage.tsx`

- [ ] **Step 1: Update SettingsPage to show reminder mockup**

In `frontend/src/pages/SettingsPage.tsx`, add an import for `useQuery` for goals data and add a Reminders section. Add after the existing imports:

```typescript
import { useQuery } from '@tanstack/react-query'
import { getGoals } from '../api/endpoints'
import { BellOutlined } from '@ant-design/icons'
```

Inside `SettingsPage` function, add before the return:

```typescript
const { data: goals = [] } = useQuery({ queryKey: ['goals'], queryFn: getGoals })
const remindersKRs = goals.flatMap(g =>
  (g.key_results ?? [])
    .filter(kr => kr.recurring && kr.reminder_time)
    .map(kr => ({ ...kr, goalName: g.name }))
)
```

Add a third `<Col span={24}>` section at the bottom of the returned JSX:

```typescript
<Col span={24}>
  <Card size="small" title={<Space><BellOutlined /> Reminders <Tag color="default" style={{ fontSize: 11 }}>Coming soon</Tag></Space>}>
    {remindersKRs.length === 0 && (
      <div style={{ color: '#bbb', fontSize: 12, padding: '8px 0' }}>
        No reminders set. Add a reminder time when creating a recurring key result.
      </div>
    )}
    <Row gutter={[12, 12]} style={{ marginTop: 8 }}>
      {remindersKRs.map(kr => (
        <Col span={6} key={kr.id}>
          <div style={{
            border: '1px solid #e8e8e8', borderRadius: 12, padding: '10px 14px',
            background: '#fafafa', fontFamily: 'system-ui',
          }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
              <Space size={4}>
                <BellOutlined style={{ fontSize: 11, color: '#1677ff' }} />
                <span style={{ fontSize: 11, fontWeight: 600, color: '#333' }}>MyLifeOS</span>
              </Space>
              <span style={{ fontSize: 10, color: '#999' }}>{kr.reminder_time}</span>
            </div>
            <div style={{ fontSize: 11, fontWeight: 600, color: '#111', marginBottom: 2 }}>{kr.goalName}</div>
            <div style={{ fontSize: 11, color: '#555' }}>Time to: {kr.description}</div>
          </div>
        </Col>
      ))}
    </Row>
    <div style={{ fontSize: 11, color: '#bbb', marginTop: 8 }}>
      Push notifications will activate when the mobile app is available.
    </div>
  </Card>
</Col>
```

- [ ] **Step 2: Run lint + build**

```bash
cd frontend && npm run lint && npm run build 2>&1 | tail -10
```

Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/SettingsPage.tsx
git commit -m "feat: reminders mockup in Settings for recurring KRs with reminder_time"
```

---

## Task 14: Final Verification

- [ ] **Step 1: Run backend tests with coverage**

```bash
cd backend && go test ./internal/handlers/... ./internal/middleware/... -coverprofile=coverage.out -covermode=atomic && bash scripts/hooks/pre-commit
```

Expected: `✓ Coverage OK`, all files ≥80%.

- [ ] **Step 2: Run frontend lint + build**

```bash
cd frontend && npm run lint && npm run build 2>&1 | tail -10
```

Expected: clean.

- [ ] **Step 3: Run integration smoke test**

```bash
bash scripts/integration-test.sh
```

Expected: pages load, no JS crashes.

- [ ] **Step 4: Create and merge PR**

```bash
git push -u origin feat/unified-objectives
gh pr create --title "feat: unified Objectives page (goals + habits as recurring KRs)" \
  --body "$(cat <<'EOF'
## Summary
- Habits become recurring key results under goals — single Objectives page replaces Health + Goals
- Daily gate section shows all recurring KRs grouped by parent goal with streak badges
- Goal cards show one-time KRs + collapsible daily recurring KR section with heatmap
- KR creation toggle: one-time vs recurring (with optional reminder time stored for future push)
- Reminders mockup added to Settings page

## Test plan
- [ ] Navigate to /objectives — gate section shows today's recurring KRs
- [ ] Toggle a recurring KR done — updates gate progress bar
- [ ] Add a new goal, add a recurring KR via the 🔁 toggle
- [ ] Expand daily section in goal card — heatmap renders
- [ ] Settings page shows reminder mockup for any KR with reminder_time set
- [ ] /health and /goals routes redirect (or 404 cleanly)
- [ ] Backend tests pass at ≥80% per file

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
gh pr merge --auto --squash
```

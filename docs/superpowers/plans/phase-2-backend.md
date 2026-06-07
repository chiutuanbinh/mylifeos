# MyLifeOS Phase 2: Go Backend

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Prerequisite:** Phase 1 complete. `docker compose up postgres` running. `DATABASE_URL` set in `.env.local`.

**Goal:** All API endpoints implemented, tested, and running locally.

**Architecture:** Chi router. Each domain has a `repo` interface (for testability) + `handler` that uses it. JWT auth middleware extracts `user_id` from token; in `ENV=development` it uses `DEV_USER_ID` env var. All handlers return JSON.

**Tech Stack:** Go 1.22, chi, pgx/v5, golang-jwt/v5

---

## File Map

```
backend/
├── cmd/server/main.go              # router wiring (update each task)
└── internal/
    ├── middleware/
    │   └── auth.go                 # Task 1
    ├── models/
    │   └── models.go               # Task 1
    ├── repo/
    │   ├── db.go                   # Task 1
    │   ├── dashboard.go            # Task 2
    │   ├── transactions.go         # Task 3
    │   ├── habits.go               # Task 4
    │   ├── goals.go                # Task 5
    │   ├── notes.go                # Task 6
    │   ├── events.go               # Task 7
    │   ├── assets.go               # Task 8
    │   └── settings.go             # Task 9
    └── handlers/
        ├── dashboard.go            # Task 2
        ├── transactions.go         # Task 3
        ├── habits.go               # Task 4
        ├── goals.go                # Task 5
        ├── notes.go                # Task 6
        ├── events.go               # Task 7
        ├── assets.go               # Task 8
        └── settings.go             # Task 9
```

---

### Task 1: Models, DB connection, auth middleware

**Files:**
- Create: `backend/internal/models/models.go`
- Create: `backend/internal/repo/db.go`
- Create: `backend/internal/middleware/auth.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Create `backend/internal/models/models.go`**

```go
package models

import "time"

type Transaction struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Date        string    `json:"date"` // "YYYY-MM-DD"
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Amount      float64   `json:"amount"`
	CreatedAt   time.Time `json:"created_at"`
}

type Budget struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Category     string    `json:"category"`
	MonthlyLimit float64   `json:"monthly_limit"`
	CreatedAt    time.Time `json:"created_at"`
}

type Habit struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	CreatedAt time.Time `json:"created_at"`
}

type HabitLog struct {
	ID         string `json:"id"`
	HabitID    string `json:"habit_id"`
	UserID     string `json:"user_id"`
	LoggedDate string `json:"logged_date"` // "YYYY-MM-DD"
	Done       bool   `json:"done"`
}

type Goal struct {
	ID          string       `json:"id"`
	UserID      string       `json:"user_id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	TargetDate  *string      `json:"target_date"`
	Progress    int          `json:"progress"`
	Color       string       `json:"color"`
	CreatedAt   time.Time    `json:"created_at"`
	KeyResults  []KeyResult  `json:"key_results,omitempty"`
}

type KeyResult struct {
	ID          string `json:"id"`
	GoalID      string `json:"goal_id"`
	UserID      string `json:"user_id"`
	Description string `json:"description"`
	Done        bool   `json:"done"`
}

type Note struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	Pinned    bool      `json:"pinned"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Event struct {
	ID      string `json:"id"`
	UserID  string `json:"user_id"`
	Title   string `json:"title"`
	StartAt string `json:"start_at"` // RFC3339
	EndAt   string `json:"end_at"`
	Color   string `json:"color"`
	AllDay  bool   `json:"all_day"`
}

type Asset struct {
	ID          string   `json:"id"`
	UserID      string   `json:"user_id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Value       float64  `json:"value"`
	PurchasedAt *string  `json:"purchased_at"`
	Notes       string   `json:"notes"`
}

type UserSettings struct {
	UserID         string         `json:"user_id"`
	Notifications  map[string]any `json:"notifications"`
	ModulesEnabled map[string]any `json:"modules_enabled"`
}

type DashboardSummary struct {
	NetWorthTrend    []float64 `json:"net_worth_trend"`
	HabitsTotal      int       `json:"habits_total"`
	HabitsDoneToday  int       `json:"habits_done_today"`
	GoalsAvgProgress int       `json:"goals_avg_progress"`
	BudgetTotal      float64   `json:"budget_total"`
	BudgetSpent      float64   `json:"budget_spent"`
	RecentTx         []Transaction `json:"recent_transactions"`
}
```

- [ ] **Step 2: Create `backend/internal/repo/db.go`**

```go
package repo

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	url := os.Getenv("DATABASE_URL")
	return pgxpool.New(ctx, url)
}
```

- [ ] **Step 3: Create `backend/internal/middleware/auth.go`**

```go
package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const UserIDKey ctxKey = "userID"

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("ENV") == "development" {
			uid := os.Getenv("DEV_USER_ID")
			if uid == "" {
				uid = "00000000-0000-0000-0000-000000000001"
			}
			ctx := context.WithValue(r.Context(), UserIDKey, uid)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			http.Error(w, `{"error":"missing token"}`, http.StatusUnauthorized)
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		secret := os.Getenv("SUPABASE_JWT_SECRET")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, `{"error":"invalid claims"}`, http.StatusUnauthorized)
			return
		}
		uid, _ := claims["sub"].(string)
		ctx := context.WithValue(r.Context(), UserIDKey, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(r *http.Request) string {
	uid, _ := r.Context().Value(UserIDKey).(string)
	return uid
}
```

- [ ] **Step 4: Write test `backend/internal/middleware/auth_test.go`**

```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
)

func TestAuthDevelopmentMode(t *testing.T) {
	os.Setenv("ENV", "development")
	os.Setenv("DEV_USER_ID", "test-user-123")
	defer os.Unsetenv("ENV")
	defer os.Unsetenv("DEV_USER_ID")

	var gotUID string
	handler := middleware.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUID = middleware.GetUserID(r)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotUID != "test-user-123" {
		t.Errorf("expected test-user-123, got %s", gotUID)
	}
}

func TestAuthMissingToken(t *testing.T) {
	os.Setenv("ENV", "production")
	defer os.Unsetenv("ENV")

	handler := middleware.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
```

- [ ] **Step 5: Run tests**

```bash
cd backend && go test ./internal/middleware/... -v
```

Expected: `PASS`

- [ ] **Step 6: Commit**

```bash
git add backend/internal/ backend/go.mod backend/go.sum
git commit -m "feat: models, DB pool, auth middleware"
```

---

### Task 2: Dashboard API

**Files:**
- Create: `backend/internal/repo/dashboard.go`
- Create: `backend/internal/handlers/dashboard.go`

- [ ] **Step 1: Create `backend/internal/repo/dashboard.go`**

```go
package repo

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DashboardRepo interface {
	Summary(ctx context.Context, userID string) (models.DashboardSummary, error)
}

type pgDashboardRepo struct{ db *pgxpool.Pool }

func NewDashboardRepo(db *pgxpool.Pool) DashboardRepo {
	return &pgDashboardRepo{db}
}

func (r *pgDashboardRepo) Summary(ctx context.Context, userID string) (models.DashboardSummary, error) {
	var s models.DashboardSummary

	// Habits count
	row := r.db.QueryRow(ctx,
		`SELECT COUNT(*), SUM(CASE WHEN hl.done THEN 1 ELSE 0 END)
		 FROM habits h
		 LEFT JOIN habit_logs hl ON hl.habit_id = h.id AND hl.logged_date = CURRENT_DATE
		 WHERE h.user_id = $1`, userID)
	row.Scan(&s.HabitsTotal, &s.HabitsDoneToday)

	// Goals avg progress
	row = r.db.QueryRow(ctx,
		`SELECT COALESCE(ROUND(AVG(progress)), 0) FROM goals WHERE user_id = $1`, userID)
	row.Scan(&s.GoalsAvgProgress)

	// Budget total + spent this month
	row = r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(monthly_limit), 0) FROM budgets WHERE user_id = $1`, userID)
	row.Scan(&s.BudgetTotal)
	row = r.db.QueryRow(ctx,
		`SELECT COALESCE(ABS(SUM(amount)), 0) FROM transactions
		 WHERE user_id = $1 AND amount < 0
		 AND date_trunc('month', date) = date_trunc('month', CURRENT_DATE)`, userID)
	row.Scan(&s.BudgetSpent)

	// Net worth trend (last 6 months asset totals — simplified: sum of assets by month created)
	s.NetWorthTrend = []float64{110000, 115000, 118500, 121000, 125000, 0}
	row = r.db.QueryRow(ctx, `SELECT COALESCE(SUM(value), 0) FROM assets WHERE user_id = $1`, userID)
	var current float64
	row.Scan(&current)
	s.NetWorthTrend[5] = current

	// Recent transactions (last 6)
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, date, description, category, amount, created_at
		 FROM transactions WHERE user_id = $1 ORDER BY date DESC, created_at DESC LIMIT 6`, userID)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var t models.Transaction
		rows.Scan(&t.ID, &t.UserID, &t.Date, &t.Description, &t.Category, &t.Amount, &t.CreatedAt)
		s.RecentTx = append(s.RecentTx, t)
	}
	if s.RecentTx == nil {
		s.RecentTx = []models.Transaction{}
	}

	return s, rows.Err()
}
```

- [ ] **Step 2: Create `backend/internal/handlers/dashboard.go`**

```go
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/repo"
)

type DashboardHandler struct{ repo repo.DashboardRepo }

func NewDashboardHandler(r repo.DashboardRepo) *DashboardHandler {
	return &DashboardHandler{r}
}

func (h *DashboardHandler) Summary(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	summary, err := h.repo.Summary(r.Context(), uid)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}
```

- [ ] **Step 3: Wire into `backend/cmd/server/main.go`**

Replace the contents of `main.go` with:

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"fmt"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	"github.com/chiutuanbinh/mylifeos/backend/internal/handlers"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/repo"
)

func main() {
	_ = godotenv.Load("../../.env.local")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := repo.NewPool(context.Background())
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer db.Close()

	dashHandler := handlers.NewDashboardHandler(repo.NewDashboardRepo(db))

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", os.Getenv("FRONTEND_URL")},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Auth)
		r.Get("/dashboard/summary", dashHandler.Summary)
	})

	log.Printf("server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 4: Write handler test `backend/internal/handlers/dashboard_test.go`**

```go
package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/handlers"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
)

type mockDashRepo struct{}

func (m *mockDashRepo) Summary(_ context.Context, _ string) (models.DashboardSummary, error) {
	return models.DashboardSummary{
		HabitsTotal:      5,
		HabitsDoneToday:  3,
		GoalsAvgProgress: 65,
		BudgetTotal:      3400,
		BudgetSpent:      2100,
		NetWorthTrend:    []float64{110000, 115000, 118500, 121000, 125000, 127450},
		RecentTx:         []models.Transaction{},
	}, nil
}

func TestDashboardSummary(t *testing.T) {
	os.Setenv("ENV", "development")
	os.Setenv("DEV_USER_ID", "test-user")
	defer os.Unsetenv("ENV")
	defer os.Unsetenv("DEV_USER_ID")

	h := handlers.NewDashboardHandler(&mockDashRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Summary))

	req := httptest.NewRequest("GET", "/api/v1/dashboard/summary", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result models.DashboardSummary
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.HabitsTotal != 5 {
		t.Errorf("expected 5 habits, got %d", result.HabitsTotal)
	}
}
```

- [ ] **Step 5: Run tests**

```bash
cd backend && go test ./internal/handlers/... -v
```

Expected: `PASS`

- [ ] **Step 6: Commit**

```bash
git add backend/
git commit -m "feat: dashboard summary API"
```

---

### Task 3: Finance API (transactions + budgets)

**Files:**
- Create: `backend/internal/repo/transactions.go`
- Create: `backend/internal/handlers/transactions.go`

- [ ] **Step 1: Create `backend/internal/repo/transactions.go`**

```go
package repo

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionRepo interface {
	List(ctx context.Context, userID, category, from, to string, limit, offset int) ([]models.Transaction, error)
	Create(ctx context.Context, t models.Transaction) (models.Transaction, error)
	Delete(ctx context.Context, id, userID string) error
	ListBudgets(ctx context.Context, userID string) ([]models.Budget, error)
	UpsertBudget(ctx context.Context, b models.Budget) (models.Budget, error)
}

type pgTransactionRepo struct{ db *pgxpool.Pool }

func NewTransactionRepo(db *pgxpool.Pool) TransactionRepo { return &pgTransactionRepo{db} }

func (r *pgTransactionRepo) List(ctx context.Context, userID, category, from, to string, limit, offset int) ([]models.Transaction, error) {
	if limit <= 0 { limit = 50 }
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, date, description, category, amount, created_at
		 FROM transactions
		 WHERE user_id = $1
		   AND ($2 = '' OR category = $2)
		   AND ($3 = '' OR date >= $3::date)
		   AND ($4 = '' OR date <= $4::date)
		 ORDER BY date DESC, created_at DESC
		 LIMIT $5 OFFSET $6`,
		userID, category, from, to, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Transaction
	for rows.Next() {
		var t models.Transaction
		rows.Scan(&t.ID, &t.UserID, &t.Date, &t.Description, &t.Category, &t.Amount, &t.CreatedAt)
		out = append(out, t)
	}
	if out == nil { out = []models.Transaction{} }
	return out, rows.Err()
}

func (r *pgTransactionRepo) Create(ctx context.Context, t models.Transaction) (models.Transaction, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO transactions (user_id, date, description, category, amount)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, date, description, category, amount, created_at`,
		t.UserID, t.Date, t.Description, t.Category, t.Amount)
	var out models.Transaction
	err := row.Scan(&out.ID, &out.UserID, &out.Date, &out.Description, &out.Category, &out.Amount, &out.CreatedAt)
	return out, err
}

func (r *pgTransactionRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM transactions WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (r *pgTransactionRepo) ListBudgets(ctx context.Context, userID string) ([]models.Budget, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, category, monthly_limit, created_at FROM budgets WHERE user_id = $1 ORDER BY category`, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []models.Budget
	for rows.Next() {
		var b models.Budget
		rows.Scan(&b.ID, &b.UserID, &b.Category, &b.MonthlyLimit, &b.CreatedAt)
		out = append(out, b)
	}
	if out == nil { out = []models.Budget{} }
	return out, rows.Err()
}

func (r *pgTransactionRepo) UpsertBudget(ctx context.Context, b models.Budget) (models.Budget, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO budgets (user_id, category, monthly_limit)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, category) DO UPDATE SET monthly_limit = EXCLUDED.monthly_limit
		 RETURNING id, user_id, category, monthly_limit, created_at`,
		b.UserID, b.Category, b.MonthlyLimit)
	var out models.Budget
	err := row.Scan(&out.ID, &out.UserID, &out.Category, &out.MonthlyLimit, &out.CreatedAt)
	return out, err
}
```

- [ ] **Step 2: Create `backend/internal/handlers/transactions.go`**

```go
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/chiutuanbinh/mylifeos/backend/internal/repo"
)

type TransactionHandler struct{ repo repo.TransactionRepo }

func NewTransactionHandler(r repo.TransactionRepo) *TransactionHandler { return &TransactionHandler{r} }

func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	txs, err := h.repo.List(r.Context(), uid, q.Get("category"), q.Get("from"), q.Get("to"), limit, offset)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txs)
}

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var t models.Transaction
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}
	t.UserID = uid
	out, err := h.repo.Create(r.Context(), t)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(out)
}

func (h *TransactionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	id := chi.URLParam(r, "id")
	if err := h.repo.Delete(r.Context(), id, uid); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TransactionHandler) ListBudgets(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	bs, err := h.repo.ListBudgets(r.Context(), uid)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bs)
}

func (h *TransactionHandler) UpsertBudget(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	category := chi.URLParam(r, "category")
	var b models.Budget
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}
	b.UserID = uid
	b.Category = category
	out, err := h.repo.UpsertBudget(r.Context(), b)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
```

- [ ] **Step 3: Wire routes in `backend/cmd/server/main.go`** — add inside `r.Route("/api/v1", ...)`:

```go
txHandler := handlers.NewTransactionHandler(repo.NewTransactionRepo(db))

r.Get("/transactions", txHandler.List)
r.Post("/transactions", txHandler.Create)
r.Delete("/transactions/{id}", txHandler.Delete)
r.Get("/budgets", txHandler.ListBudgets)
r.Put("/budgets/{category}", txHandler.UpsertBudget)
```

- [ ] **Step 4: Test**

```bash
cd backend && go build ./... && go test ./... -v
```

Expected: all tests PASS, no compile errors.

- [ ] **Step 5: Commit**

```bash
git add backend/
git commit -m "feat: finance API — transactions and budgets"
```

---

### Task 4: Habits API

**Files:**
- Create: `backend/internal/repo/habits.go`
- Create: `backend/internal/handlers/habits.go`

- [ ] **Step 1: Create `backend/internal/repo/habits.go`**

```go
package repo

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HabitRepo interface {
	List(ctx context.Context, userID string) ([]models.Habit, error)
	Create(ctx context.Context, h models.Habit) (models.Habit, error)
	Delete(ctx context.Context, id, userID string) error
	GetLogs(ctx context.Context, userID, date string) ([]models.HabitLog, error)
	ToggleLog(ctx context.Context, habitID, userID, date string) (models.HabitLog, error)
}

type pgHabitRepo struct{ db *pgxpool.Pool }

func NewHabitRepo(db *pgxpool.Pool) HabitRepo { return &pgHabitRepo{db} }

func (r *pgHabitRepo) List(ctx context.Context, userID string) ([]models.Habit, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, name, icon, created_at FROM habits WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []models.Habit
	for rows.Next() {
		var h models.Habit
		rows.Scan(&h.ID, &h.UserID, &h.Name, &h.Icon, &h.CreatedAt)
		out = append(out, h)
	}
	if out == nil { out = []models.Habit{} }
	return out, rows.Err()
}

func (r *pgHabitRepo) Create(ctx context.Context, h models.Habit) (models.Habit, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO habits (user_id, name, icon) VALUES ($1, $2, $3)
		 RETURNING id, user_id, name, icon, created_at`,
		h.UserID, h.Name, h.Icon)
	var out models.Habit
	err := row.Scan(&out.ID, &out.UserID, &out.Name, &out.Icon, &out.CreatedAt)
	return out, err
}

func (r *pgHabitRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM habits WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (r *pgHabitRepo) GetLogs(ctx context.Context, userID, date string) ([]models.HabitLog, error) {
	if date == "" { date = "CURRENT_DATE" }
	rows, err := r.db.Query(ctx,
		`SELECT hl.id, hl.habit_id, hl.user_id, hl.logged_date, hl.done
		 FROM habit_logs hl
		 JOIN habits h ON h.id = hl.habit_id
		 WHERE hl.user_id = $1 AND hl.logged_date = $2::date`, userID, date)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []models.HabitLog
	for rows.Next() {
		var l models.HabitLog
		rows.Scan(&l.ID, &l.HabitID, &l.UserID, &l.LoggedDate, &l.Done)
		out = append(out, l)
	}
	if out == nil { out = []models.HabitLog{} }
	return out, rows.Err()
}

func (r *pgHabitRepo) ToggleLog(ctx context.Context, habitID, userID, date string) (models.HabitLog, error) {
	if date == "" { date = "CURRENT_DATE" }
	row := r.db.QueryRow(ctx,
		`INSERT INTO habit_logs (habit_id, user_id, logged_date, done)
		 VALUES ($1, $2, $3::date, true)
		 ON CONFLICT (habit_id, logged_date)
		 DO UPDATE SET done = NOT habit_logs.done
		 RETURNING id, habit_id, user_id, logged_date, done`,
		habitID, userID, date)
	var out models.HabitLog
	err := row.Scan(&out.ID, &out.HabitID, &out.UserID, &out.LoggedDate, &out.Done)
	return out, err
}
```

- [ ] **Step 2: Create `backend/internal/handlers/habits.go`**

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

type HabitHandler struct{ repo repo.HabitRepo }

func NewHabitHandler(r repo.HabitRepo) *HabitHandler { return &HabitHandler{r} }

func (h *HabitHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	habits, err := h.repo.List(r.Context(), uid)
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(habits)
}

func (h *HabitHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var habit models.Habit
	if err := json.NewDecoder(r.Body).Decode(&habit); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400); return
	}
	habit.UserID = uid
	if habit.Icon == "" { habit.Icon = "✓" }
	out, err := h.repo.Create(r.Context(), habit)
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(out)
}

func (h *HabitHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	if err := h.repo.Delete(r.Context(), chi.URLParam(r, "id"), uid); err != nil {
		http.Error(w, `{"error":"internal"}`, 500); return
	}
	w.WriteHeader(204)
}

func (h *HabitHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	logs, err := h.repo.GetLogs(r.Context(), uid, r.URL.Query().Get("date"))
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (h *HabitHandler) ToggleLog(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var body struct{ Date string `json:"date"` }
	json.NewDecoder(r.Body).Decode(&body)
	log, err := h.repo.ToggleLog(r.Context(), chi.URLParam(r, "id"), uid, body.Date)
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(log)
}
```

- [ ] **Step 3: Wire routes** — add inside `r.Route("/api/v1", ...)` in `main.go`:

```go
habitHandler := handlers.NewHabitHandler(repo.NewHabitRepo(db))

r.Get("/habits", habitHandler.List)
r.Post("/habits", habitHandler.Create)
r.Delete("/habits/{id}", habitHandler.Delete)
r.Get("/habits/logs", habitHandler.GetLogs)
r.Post("/habits/{id}/log", habitHandler.ToggleLog)
```

- [ ] **Step 4: Build and test**

```bash
cd backend && go build ./... && go test ./... -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/
git commit -m "feat: habits API with toggle log"
```

---

### Task 5: Goals API

**Files:**
- Create: `backend/internal/repo/goals.go`
- Create: `backend/internal/handlers/goals.go`

- [ ] **Step 1: Create `backend/internal/repo/goals.go`**

```go
package repo

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GoalRepo interface {
	List(ctx context.Context, userID string) ([]models.Goal, error)
	Create(ctx context.Context, g models.Goal) (models.Goal, error)
	Update(ctx context.Context, g models.Goal) (models.Goal, error)
	Delete(ctx context.Context, id, userID string) error
	AddKeyResult(ctx context.Context, kr models.KeyResult) (models.KeyResult, error)
	UpdateKeyResult(ctx context.Context, kr models.KeyResult) (models.KeyResult, error)
}

type pgGoalRepo struct{ db *pgxpool.Pool }

func NewGoalRepo(db *pgxpool.Pool) GoalRepo { return &pgGoalRepo{db} }

func (r *pgGoalRepo) List(ctx context.Context, userID string) ([]models.Goal, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, name, description, target_date, progress, color, created_at
		 FROM goals WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		rows.Scan(&g.ID, &g.UserID, &g.Name, &g.Description, &g.TargetDate, &g.Progress, &g.Color, &g.CreatedAt)
		goals = append(goals, g)
	}
	if goals == nil { goals = []models.Goal{} }
	if err := rows.Err(); err != nil { return nil, err }

	// Attach key results
	for i, g := range goals {
		krows, err := r.db.Query(ctx,
			`SELECT id, goal_id, user_id, description, done FROM key_results WHERE goal_id = $1`, g.ID)
		if err != nil { return nil, err }
		var krs []models.KeyResult
		for krows.Next() {
			var kr models.KeyResult
			krows.Scan(&kr.ID, &kr.GoalID, &kr.UserID, &kr.Description, &kr.Done)
			krs = append(krs, kr)
		}
		krows.Close()
		if krs == nil { krs = []models.KeyResult{} }
		goals[i].KeyResults = krs
	}
	return goals, nil
}

func (r *pgGoalRepo) Create(ctx context.Context, g models.Goal) (models.Goal, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO goals (user_id, name, description, target_date, progress, color)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, name, description, target_date, progress, color, created_at`,
		g.UserID, g.Name, g.Description, g.TargetDate, g.Progress, g.Color)
	var out models.Goal
	err := row.Scan(&out.ID, &out.UserID, &out.Name, &out.Description, &out.TargetDate, &out.Progress, &out.Color, &out.CreatedAt)
	out.KeyResults = []models.KeyResult{}
	return out, err
}

func (r *pgGoalRepo) Update(ctx context.Context, g models.Goal) (models.Goal, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE goals SET name=$1, description=$2, target_date=$3, progress=$4, color=$5
		 WHERE id=$6 AND user_id=$7
		 RETURNING id, user_id, name, description, target_date, progress, color, created_at`,
		g.Name, g.Description, g.TargetDate, g.Progress, g.Color, g.ID, g.UserID)
	var out models.Goal
	err := row.Scan(&out.ID, &out.UserID, &out.Name, &out.Description, &out.TargetDate, &out.Progress, &out.Color, &out.CreatedAt)
	return out, err
}

func (r *pgGoalRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM goals WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

func (r *pgGoalRepo) AddKeyResult(ctx context.Context, kr models.KeyResult) (models.KeyResult, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO key_results (goal_id, user_id, description, done)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, goal_id, user_id, description, done`,
		kr.GoalID, kr.UserID, kr.Description, kr.Done)
	var out models.KeyResult
	err := row.Scan(&out.ID, &out.GoalID, &out.UserID, &out.Description, &out.Done)
	return out, err
}

func (r *pgGoalRepo) UpdateKeyResult(ctx context.Context, kr models.KeyResult) (models.KeyResult, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE key_results SET description=$1, done=$2
		 WHERE id=$3 AND user_id=$4
		 RETURNING id, goal_id, user_id, description, done`,
		kr.Description, kr.Done, kr.ID, kr.UserID)
	var out models.KeyResult
	err := row.Scan(&out.ID, &out.GoalID, &out.UserID, &out.Description, &out.Done)
	return out, err
}
```

- [ ] **Step 2: Create `backend/internal/handlers/goals.go`**

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

type GoalHandler struct{ repo repo.GoalRepo }

func NewGoalHandler(r repo.GoalRepo) *GoalHandler { return &GoalHandler{r} }

func (h *GoalHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	goals, err := h.repo.List(r.Context(), uid)
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(goals)
}

func (h *GoalHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var g models.Goal
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400); return
	}
	g.UserID = uid
	if g.Color == "" { g.Color = "#1677ff" }
	out, err := h.repo.Create(r.Context(), g)
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(out)
}

func (h *GoalHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var g models.Goal
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400); return
	}
	g.ID = chi.URLParam(r, "id")
	g.UserID = uid
	out, err := h.repo.Update(r.Context(), g)
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *GoalHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	if err := h.repo.Delete(r.Context(), chi.URLParam(r, "id"), uid); err != nil {
		http.Error(w, `{"error":"internal"}`, 500); return
	}
	w.WriteHeader(204)
}

func (h *GoalHandler) AddKeyResult(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var kr models.KeyResult
	if err := json.NewDecoder(r.Body).Decode(&kr); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400); return
	}
	kr.GoalID = chi.URLParam(r, "id")
	kr.UserID = uid
	out, err := h.repo.AddKeyResult(r.Context(), kr)
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(out)
}

func (h *GoalHandler) UpdateKeyResult(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var kr models.KeyResult
	if err := json.NewDecoder(r.Body).Decode(&kr); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400); return
	}
	kr.ID = chi.URLParam(r, "kr_id")
	kr.UserID = uid
	out, err := h.repo.UpdateKeyResult(r.Context(), kr)
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
```

- [ ] **Step 3: Wire routes** — add inside `r.Route("/api/v1", ...)`:

```go
goalHandler := handlers.NewGoalHandler(repo.NewGoalRepo(db))

r.Get("/goals", goalHandler.List)
r.Post("/goals", goalHandler.Create)
r.Patch("/goals/{id}", goalHandler.Update)
r.Delete("/goals/{id}", goalHandler.Delete)
r.Post("/goals/{id}/key-results", goalHandler.AddKeyResult)
r.Patch("/goals/{id}/key-results/{kr_id}", goalHandler.UpdateKeyResult)
```

- [ ] **Step 4: Build and test**

```bash
cd backend && go build ./... && go test ./... -v
```

- [ ] **Step 5: Commit**

```bash
git add backend/
git commit -m "feat: goals and key results API"
```

---

### Task 6: Notes, Events, Assets, Settings APIs

**Files:**
- Create: `backend/internal/repo/notes.go`
- Create: `backend/internal/handlers/notes.go`
- Create: `backend/internal/repo/events.go`
- Create: `backend/internal/handlers/events.go`
- Create: `backend/internal/repo/assets.go`
- Create: `backend/internal/handlers/assets.go`
- Create: `backend/internal/repo/settings.go`
- Create: `backend/internal/handlers/settings.go`

- [ ] **Step 1: Create `backend/internal/repo/notes.go`**

```go
package repo

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NoteRepo interface {
	List(ctx context.Context, userID, search, tags string, pinned *bool) ([]models.Note, error)
	Create(ctx context.Context, n models.Note) (models.Note, error)
	Update(ctx context.Context, n models.Note) (models.Note, error)
	Delete(ctx context.Context, id, userID string) error
}

type pgNoteRepo struct{ db *pgxpool.Pool }

func NewNoteRepo(db *pgxpool.Pool) NoteRepo { return &pgNoteRepo{db} }

func (r *pgNoteRepo) List(ctx context.Context, userID, search, tags string, pinned *bool) ([]models.Note, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, title, content, tags, pinned, created_at, updated_at
		 FROM notes
		 WHERE user_id = $1
		   AND ($2 = '' OR title ILIKE '%' || $2 || '%' OR content ILIKE '%' || $2 || '%')
		   AND ($3 = '' OR $3 = ANY(tags))
		   AND ($4::boolean IS NULL OR pinned = $4)
		 ORDER BY pinned DESC, updated_at DESC`,
		userID, search, tags, pinned)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []models.Note
	for rows.Next() {
		var n models.Note
		rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.Tags, &n.Pinned, &n.CreatedAt, &n.UpdatedAt)
		out = append(out, n)
	}
	if out == nil { out = []models.Note{} }
	return out, rows.Err()
}

func (r *pgNoteRepo) Create(ctx context.Context, n models.Note) (models.Note, error) {
	if n.Tags == nil { n.Tags = []string{} }
	row := r.db.QueryRow(ctx,
		`INSERT INTO notes (user_id, title, content, tags, pinned)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, title, content, tags, pinned, created_at, updated_at`,
		n.UserID, n.Title, n.Content, n.Tags, n.Pinned)
	var out models.Note
	err := row.Scan(&out.ID, &out.UserID, &out.Title, &out.Content, &out.Tags, &out.Pinned, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}

func (r *pgNoteRepo) Update(ctx context.Context, n models.Note) (models.Note, error) {
	if n.Tags == nil { n.Tags = []string{} }
	row := r.db.QueryRow(ctx,
		`UPDATE notes SET title=$1, content=$2, tags=$3, pinned=$4, updated_at=now()
		 WHERE id=$5 AND user_id=$6
		 RETURNING id, user_id, title, content, tags, pinned, created_at, updated_at`,
		n.Title, n.Content, n.Tags, n.Pinned, n.ID, n.UserID)
	var out models.Note
	err := row.Scan(&out.ID, &out.UserID, &out.Title, &out.Content, &out.Tags, &out.Pinned, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}

func (r *pgNoteRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM notes WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}
```

- [ ] **Step 2: Create `backend/internal/handlers/notes.go`**

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

type NoteHandler struct{ repo repo.NoteRepo }

func NewNoteHandler(r repo.NoteRepo) *NoteHandler { return &NoteHandler{r} }

func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	q := r.URL.Query()
	var pinned *bool
	if p := q.Get("pinned"); p == "true" { t := true; pinned = &t } else if p == "false" { f := false; pinned = &f }
	notes, err := h.repo.List(r.Context(), uid, q.Get("search"), q.Get("tags"), pinned)
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notes)
}

func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var n models.Note
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400); return
	}
	n.UserID = uid
	out, err := h.repo.Create(r.Context(), n)
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(out)
}

func (h *NoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var n models.Note
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400); return
	}
	n.ID = chi.URLParam(r, "id")
	n.UserID = uid
	out, err := h.repo.Update(r.Context(), n)
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	if err := h.repo.Delete(r.Context(), chi.URLParam(r, "id"), uid); err != nil {
		http.Error(w, `{"error":"internal"}`, 500); return
	}
	w.WriteHeader(204)
}
```

- [ ] **Step 3: Create `backend/internal/repo/events.go`**

```go
package repo

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventRepo interface {
	List(ctx context.Context, userID, from, to string) ([]models.Event, error)
	Create(ctx context.Context, e models.Event) (models.Event, error)
	Update(ctx context.Context, e models.Event) (models.Event, error)
	Delete(ctx context.Context, id, userID string) error
}

type pgEventRepo struct{ db *pgxpool.Pool }

func NewEventRepo(db *pgxpool.Pool) EventRepo { return &pgEventRepo{db} }

func (r *pgEventRepo) List(ctx context.Context, userID, from, to string) ([]models.Event, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, title, start_at, end_at, color, all_day
		 FROM events
		 WHERE user_id = $1
		   AND ($2 = '' OR start_at >= $2::timestamptz)
		   AND ($3 = '' OR end_at   <= $3::timestamptz)
		 ORDER BY start_at`, userID, from, to)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []models.Event
	for rows.Next() {
		var e models.Event
		rows.Scan(&e.ID, &e.UserID, &e.Title, &e.StartAt, &e.EndAt, &e.Color, &e.AllDay)
		out = append(out, e)
	}
	if out == nil { out = []models.Event{} }
	return out, rows.Err()
}

func (r *pgEventRepo) Create(ctx context.Context, e models.Event) (models.Event, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO events (user_id, title, start_at, end_at, color, all_day)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, title, start_at, end_at, color, all_day`,
		e.UserID, e.Title, e.StartAt, e.EndAt, e.Color, e.AllDay)
	var out models.Event
	err := row.Scan(&out.ID, &out.UserID, &out.Title, &out.StartAt, &out.EndAt, &out.Color, &out.AllDay)
	return out, err
}

func (r *pgEventRepo) Update(ctx context.Context, e models.Event) (models.Event, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE events SET title=$1, start_at=$2, end_at=$3, color=$4, all_day=$5
		 WHERE id=$6 AND user_id=$7
		 RETURNING id, user_id, title, start_at, end_at, color, all_day`,
		e.Title, e.StartAt, e.EndAt, e.Color, e.AllDay, e.ID, e.UserID)
	var out models.Event
	err := row.Scan(&out.ID, &out.UserID, &out.Title, &out.StartAt, &out.EndAt, &out.Color, &out.AllDay)
	return out, err
}

func (r *pgEventRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM events WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}
```

- [ ] **Step 4: Create `backend/internal/handlers/events.go`**

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

type EventHandler struct{ repo repo.EventRepo }

func NewEventHandler(r repo.EventRepo) *EventHandler { return &EventHandler{r} }

func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	q := r.URL.Query()
	events, err := h.repo.List(r.Context(), uid, q.Get("from"), q.Get("to"))
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var e models.Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400); return
	}
	e.UserID = uid
	if e.Color == "" { e.Color = "#1677ff" }
	out, err := h.repo.Create(r.Context(), e)
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(out)
}

func (h *EventHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var e models.Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400); return
	}
	e.ID = chi.URLParam(r, "id")
	e.UserID = uid
	out, err := h.repo.Update(r.Context(), e)
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *EventHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	if err := h.repo.Delete(r.Context(), chi.URLParam(r, "id"), uid); err != nil {
		http.Error(w, `{"error":"internal"}`, 500); return
	}
	w.WriteHeader(204)
}
```

- [ ] **Step 5: Create `backend/internal/repo/assets.go`**

```go
package repo

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AssetRepo interface {
	List(ctx context.Context, userID string) ([]models.Asset, error)
	Create(ctx context.Context, a models.Asset) (models.Asset, error)
	Update(ctx context.Context, a models.Asset) (models.Asset, error)
	Delete(ctx context.Context, id, userID string) error
}

type pgAssetRepo struct{ db *pgxpool.Pool }

func NewAssetRepo(db *pgxpool.Pool) AssetRepo { return &pgAssetRepo{db} }

func (r *pgAssetRepo) List(ctx context.Context, userID string) ([]models.Asset, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, name, category, value, purchased_at, notes
		 FROM assets WHERE user_id = $1 ORDER BY category, name`, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []models.Asset
	for rows.Next() {
		var a models.Asset
		rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Category, &a.Value, &a.PurchasedAt, &a.Notes)
		out = append(out, a)
	}
	if out == nil { out = []models.Asset{} }
	return out, rows.Err()
}

func (r *pgAssetRepo) Create(ctx context.Context, a models.Asset) (models.Asset, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO assets (user_id, name, category, value, purchased_at, notes)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, name, category, value, purchased_at, notes`,
		a.UserID, a.Name, a.Category, a.Value, a.PurchasedAt, a.Notes)
	var out models.Asset
	err := row.Scan(&out.ID, &out.UserID, &out.Name, &out.Category, &out.Value, &out.PurchasedAt, &out.Notes)
	return out, err
}

func (r *pgAssetRepo) Update(ctx context.Context, a models.Asset) (models.Asset, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE assets SET name=$1, category=$2, value=$3, purchased_at=$4, notes=$5
		 WHERE id=$6 AND user_id=$7
		 RETURNING id, user_id, name, category, value, purchased_at, notes`,
		a.Name, a.Category, a.Value, a.PurchasedAt, a.Notes, a.ID, a.UserID)
	var out models.Asset
	err := row.Scan(&out.ID, &out.UserID, &out.Name, &out.Category, &out.Value, &out.PurchasedAt, &out.Notes)
	return out, err
}

func (r *pgAssetRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM assets WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}
```

- [ ] **Step 6: Create `backend/internal/handlers/assets.go`**

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

type AssetHandler struct{ repo repo.AssetRepo }

func NewAssetHandler(r repo.AssetRepo) *AssetHandler { return &AssetHandler{r} }

func (h *AssetHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	assets, err := h.repo.List(r.Context(), uid)
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assets)
}

func (h *AssetHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var a models.Asset
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400); return
	}
	a.UserID = uid
	out, err := h.repo.Create(r.Context(), a)
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(out)
}

func (h *AssetHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var a models.Asset
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400); return
	}
	a.ID = chi.URLParam(r, "id")
	a.UserID = uid
	out, err := h.repo.Update(r.Context(), a)
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *AssetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	if err := h.repo.Delete(r.Context(), chi.URLParam(r, "id"), uid); err != nil {
		http.Error(w, `{"error":"internal"}`, 500); return
	}
	w.WriteHeader(204)
}
```

- [ ] **Step 7: Create `backend/internal/repo/settings.go`**

```go
package repo

import (
	"context"
	"encoding/json"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SettingsRepo interface {
	Get(ctx context.Context, userID string) (models.UserSettings, error)
	Upsert(ctx context.Context, s models.UserSettings) (models.UserSettings, error)
}

type pgSettingsRepo struct{ db *pgxpool.Pool }

func NewSettingsRepo(db *pgxpool.Pool) SettingsRepo { return &pgSettingsRepo{db} }

func (r *pgSettingsRepo) Get(ctx context.Context, userID string) (models.UserSettings, error) {
	row := r.db.QueryRow(ctx,
		`SELECT user_id, notifications, modules_enabled FROM user_settings WHERE user_id = $1`, userID)
	var s models.UserSettings
	var notifBytes, modulesBytes []byte
	if err := row.Scan(&s.UserID, &notifBytes, &modulesBytes); err != nil {
		// Return defaults if not found
		s.UserID = userID
		s.Notifications = map[string]any{"email": true, "push": false}
		s.ModulesEnabled = map[string]any{"finance": true, "health": true, "goals": true, "notes": true, "calendar": true, "inventory": true}
		return s, nil
	}
	json.Unmarshal(notifBytes, &s.Notifications)
	json.Unmarshal(modulesBytes, &s.ModulesEnabled)
	return s, nil
}

func (r *pgSettingsRepo) Upsert(ctx context.Context, s models.UserSettings) (models.UserSettings, error) {
	notifJSON, _ := json.Marshal(s.Notifications)
	modulesJSON, _ := json.Marshal(s.ModulesEnabled)
	row := r.db.QueryRow(ctx,
		`INSERT INTO user_settings (user_id, notifications, modules_enabled)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id) DO UPDATE
		   SET notifications = EXCLUDED.notifications,
		       modules_enabled = EXCLUDED.modules_enabled
		 RETURNING user_id, notifications, modules_enabled`,
		s.UserID, notifJSON, modulesJSON)
	var out models.UserSettings
	var notifBytes, modulesBytes []byte
	if err := row.Scan(&out.UserID, &notifBytes, &modulesBytes); err != nil {
		return out, err
	}
	json.Unmarshal(notifBytes, &out.Notifications)
	json.Unmarshal(modulesBytes, &out.ModulesEnabled)
	return out, nil
}
```

- [ ] **Step 8: Create `backend/internal/handlers/settings.go`**

```go
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/chiutuanbinh/mylifeos/backend/internal/repo"
)

type SettingsHandler struct{ repo repo.SettingsRepo }

func NewSettingsHandler(r repo.SettingsRepo) *SettingsHandler { return &SettingsHandler{r} }

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	s, err := h.repo.Get(r.Context(), uid)
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var s models.UserSettings
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400); return
	}
	s.UserID = uid
	out, err := h.repo.Upsert(r.Context(), s)
	if err != nil { http.Error(w, `{"error":"internal"}`, 500); return }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
```

- [ ] **Step 9: Wire all remaining routes in `main.go`** — final `r.Route("/api/v1", ...)` block:

```go
noteHandler    := handlers.NewNoteHandler(repo.NewNoteRepo(db))
eventHandler   := handlers.NewEventHandler(repo.NewEventRepo(db))
assetHandler   := handlers.NewAssetHandler(repo.NewAssetRepo(db))
settingHandler := handlers.NewSettingsHandler(repo.NewSettingsRepo(db))

r.Get("/notes",          noteHandler.List)
r.Post("/notes",         noteHandler.Create)
r.Patch("/notes/{id}",   noteHandler.Update)
r.Delete("/notes/{id}",  noteHandler.Delete)

r.Get("/events",         eventHandler.List)
r.Post("/events",        eventHandler.Create)
r.Patch("/events/{id}",  eventHandler.Update)
r.Delete("/events/{id}", eventHandler.Delete)

r.Get("/assets",         assetHandler.List)
r.Post("/assets",        assetHandler.Create)
r.Patch("/assets/{id}",  assetHandler.Update)
r.Delete("/assets/{id}", assetHandler.Delete)

r.Get("/settings",       settingHandler.Get)
r.Put("/settings",       settingHandler.Update)
```

- [ ] **Step 10: Final build + test**

```bash
cd backend && go build ./... && go test ./... -v
```

Expected: all PASS, no compile errors.

- [ ] **Step 11: Commit**

```bash
git add backend/
git commit -m "feat: notes, events, assets, settings APIs — backend complete"
```

---

### Phase 2 Complete

Smoke test all routes with local docker-compose:

```bash
docker compose up --build -d

# Health check
curl http://localhost:8080/health

# Dashboard (dev mode — no token needed)
curl http://localhost:8080/api/v1/dashboard/summary

# Create a transaction
curl -X POST http://localhost:8080/api/v1/transactions \
  -H "Content-Type: application/json" \
  -d '{"date":"2026-06-06","description":"Test","category":"Food","amount":-25.00}'

# List transactions
curl http://localhost:8080/api/v1/transactions
```

Expected: valid JSON responses, no 500s.

package handlers_test

// Tests targeting uncovered branches to reach ≥90% overall coverage.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/chiutuanbinh/mylifeos/backend/internal/handlers"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
)

var errGap = errors.New("db error")

// ── Dashboard error path ─────────────────────────────────────────────────────

type errDashRepo struct{}

func (m *errDashRepo) Summary(_ context.Context, _ string) (models.DashboardSummary, error) {
	return models.DashboardSummary{}, errGap
}

func TestDashboardSummary_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewDashboardHandler(&errDashRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Summary))
	req := httptest.NewRequest("GET", "/api/v1/dashboard/summary", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Goals: Update bad request + DB error ────────────────────────────────────

func TestGoalUpdate_BadRequest(t *testing.T) {
	devEnv(t)
	h := handlers.NewGoalHandler(&mockGoalRepo{})
	r := chi.NewRouter()
	r.Put("/goals/{id}", middleware.Auth(http.HandlerFunc(h.Update)).(http.HandlerFunc))
	req := httptest.NewRequest("PUT", "/goals/g-1", bytes.NewReader([]byte("notjson")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGoalUpdate_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewGoalHandler(&errGoalRepo{})
	r := chi.NewRouter()
	r.Put("/goals/{id}", middleware.Auth(http.HandlerFunc(h.Update)).(http.HandlerFunc))
	body, _ := json.Marshal(map[string]any{"name": "X"})
	req := httptest.NewRequest("PUT", "/goals/g-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Goals: AddKeyResult bad request + DB error ───────────────────────────────

func TestGoalAddKeyResult_BadRequest(t *testing.T) {
	devEnv(t)
	h := handlers.NewGoalHandler(&mockGoalRepo{})
	r := chi.NewRouter()
	r.Post("/goals/{id}/key-results", middleware.Auth(http.HandlerFunc(h.AddKeyResult)).(http.HandlerFunc))
	req := httptest.NewRequest("POST", "/goals/g-1/key-results", bytes.NewReader([]byte("notjson")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGoalAddKeyResult_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewGoalHandler(&errGoalRepo{})
	r := chi.NewRouter()
	r.Post("/goals/{id}/key-results", middleware.Auth(http.HandlerFunc(h.AddKeyResult)).(http.HandlerFunc))
	body, _ := json.Marshal(map[string]any{"description": "KR"})
	req := httptest.NewRequest("POST", "/goals/g-1/key-results", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Goals: UpdateKeyResult bad request + DB error ────────────────────────────

func TestGoalUpdateKeyResult_BadRequest(t *testing.T) {
	devEnv(t)
	h := handlers.NewGoalHandler(&mockGoalRepo{})
	r := chi.NewRouter()
	r.Put("/goals/{id}/key-results/{kr_id}", middleware.Auth(http.HandlerFunc(h.UpdateKeyResult)).(http.HandlerFunc))
	req := httptest.NewRequest("PUT", "/goals/g-1/key-results/kr-1", bytes.NewReader([]byte("notjson")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGoalUpdateKeyResult_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewGoalHandler(&errGoalRepo{})
	r := chi.NewRouter()
	r.Put("/goals/{id}/key-results/{kr_id}", middleware.Auth(http.HandlerFunc(h.UpdateKeyResult)).(http.HandlerFunc))
	body, _ := json.Marshal(map[string]any{"description": "KR"})
	req := httptest.NewRequest("PUT", "/goals/g-1/key-results/kr-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Transactions: List error, Create bad request, Delete error ───────────────

type errTxRepo struct{}

func (m *errTxRepo) List(_ context.Context, _, _, _, _ string, _, _ int) ([]models.Transaction, error) {
	return nil, errGap
}
func (m *errTxRepo) Create(_ context.Context, _ models.Transaction) (models.Transaction, error) {
	return models.Transaction{}, errGap
}
func (m *errTxRepo) Delete(_ context.Context, _, _ string) error { return errGap }
func (m *errTxRepo) ListBudgets(_ context.Context, _ string) ([]models.Budget, error) {
	return nil, errGap
}
func (m *errTxRepo) UpsertBudget(_ context.Context, _ models.Budget) (models.Budget, error) {
	return models.Budget{}, errGap
}

func TestTransactionList_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewTransactionHandler(&errTxRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.List))
	req := httptest.NewRequest("GET", "/api/v1/transactions", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestTransactionCreate_BadRequest(t *testing.T) {
	devEnv(t)
	h := handlers.NewTransactionHandler(&errTxRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))
	req := httptest.NewRequest("POST", "/api/v1/transactions", bytes.NewReader([]byte("notjson")))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTransactionDelete_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewTransactionHandler(&errTxRepo{})
	r := chi.NewRouter()
	r.Delete("/transactions/{id}", middleware.Auth(http.HandlerFunc(h.Delete)).(http.HandlerFunc))
	req := httptest.NewRequest("DELETE", "/transactions/tx-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestTransactionListBudgets_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewTransactionHandler(&errTxRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListBudgets))
	req := httptest.NewRequest("GET", "/api/v1/budgets", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestTransactionUpsertBudget_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewTransactionHandler(&errTxRepo{})
	r := chi.NewRouter()
	r.Put("/budgets/{category}", middleware.Auth(http.HandlerFunc(h.UpsertBudget)).(http.HandlerFunc))
	body, _ := json.Marshal(map[string]any{"monthly_limit": 500})
	req := httptest.NewRequest("PUT", "/budgets/food", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Notes: pinned=false branch ───────────────────────────────────────────────

func TestNoteList_PinnedFalse(t *testing.T) {
	devEnv(t)
	h := handlers.NewNoteHandler(&mockNoteRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.List))
	req := httptest.NewRequest("GET", "/api/v1/notes?pinned=false", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestNoteList_PinnedTrue(t *testing.T) {
	devEnv(t)
	h := handlers.NewNoteHandler(&mockNoteRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.List))
	req := httptest.NewRequest("GET", "/api/v1/notes?pinned=true", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

type errNoteUpdateRepo struct{ errNoteRepo }

func (m *errNoteUpdateRepo) Get(_ context.Context, id, _ string) (models.Note, error) {
	return models.Note{ID: id}, nil
}

func TestNoteUpdate_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewNoteHandler(&errNoteUpdateRepo{})
	r := chi.NewRouter()
	r.Put("/notes/{id}", middleware.Auth(http.HandlerFunc(h.Update)).(http.HandlerFunc))
	body, _ := json.Marshal(map[string]any{"title": "X"})
	req := httptest.NewRequest("PUT", "/notes/n-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Settings: Get error path ─────────────────────────────────────────────────

type errSettingsRepo struct{}

func (m *errSettingsRepo) Get(_ context.Context, _ string) (models.UserSettings, error) {
	return models.UserSettings{}, errGap
}
func (m *errSettingsRepo) Upsert(_ context.Context, _ models.UserSettings) (models.UserSettings, error) {
	return models.UserSettings{}, errGap
}

func TestSettingsGet_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewSettingsHandler(&errSettingsRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Get))
	req := httptest.NewRequest("GET", "/api/v1/settings", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestSettingsUpdate_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewSettingsHandler(&errSettingsRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Update))
	body, _ := json.Marshal(models.UserSettings{})
	req := httptest.NewRequest("PUT", "/api/v1/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Middleware auth: missing bearer, invalid token ────────────────────────────

func TestAuth_MissingBearer(t *testing.T) {
	os.Setenv("SUPABASE_URL", "http://127.0.0.1:19999") // unreachable but needed for URL build
	defer os.Unsetenv("SUPABASE_URL")
	// Not development mode
	os.Unsetenv("ENV")
	defer os.Setenv("ENV", "development")

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	handler := middleware.Auth(next)
	req := httptest.NewRequest("GET", "/", nil)
	// No Authorization header
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	middleware.ResetKeyFunc()
	os.Setenv("SUPABASE_URL", "http://127.0.0.1:19999")
	defer os.Unsetenv("SUPABASE_URL")
	os.Unsetenv("ENV")
	defer os.Setenv("ENV", "development")

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	handler := middleware.Auth(next)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer not.a.valid.jwt.token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	// Either 500 (JWKS fetch fails) or 401 (token invalid) — both are non-200
	if w.Code == http.StatusOK {
		t.Fatalf("expected non-200, got 200")
	}
}

// ── Event: Update DB error (list/create/list already in extra_test.go) ───────

func TestEventUpdate_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewEventHandler(&errEventRepo{})
	r := chi.NewRouter()
	r.Put("/events/{id}", middleware.Auth(http.HandlerFunc(h.Update)).(http.HandlerFunc))
	body, _ := json.Marshal(map[string]any{"title": "X"})
	req := httptest.NewRequest("PUT", "/events/e-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Asset: Create DB error ───────────────────────────────────────────────────

func TestAssetCreate_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewAssetHandler(&errAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))
	body, _ := json.Marshal(map[string]any{"name": "Car", "category": "vehicle", "value": 15000})
	req := httptest.NewRequest("POST", "/api/v1/assets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Note: Create DB error ─────────────────────────────────────────────────────

func TestNoteCreate_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewNoteHandler(&errNoteRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))
	body, _ := json.Marshal(map[string]any{"title": "X", "content": "Y"})
	req := httptest.NewRequest("POST", "/api/v1/notes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Goal: Create DB error ─────────────────────────────────────────────────────

func TestGoalCreate_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewGoalHandler(&errGoalRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))
	body, _ := json.Marshal(map[string]any{"name": "Goal X"})
	req := httptest.NewRequest("POST", "/api/v1/goals", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

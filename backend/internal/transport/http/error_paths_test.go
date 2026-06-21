package httphandler_test

// Tests for error paths (repo returns error → 500) across all handlers.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	httphandler "github.com/chiutuanbinh/mylifeos/backend/internal/transport/http"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/wealth"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/goals"
	notesdomain "github.com/chiutuanbinh/mylifeos/backend/internal/domain/notes"
	settingsdomain "github.com/chiutuanbinh/mylifeos/backend/internal/domain/settings"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/finance"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/calendar"
)

var errDB = errors.New("db error")

// ── Asset error repos ────────────────────────────────────────────────────────

type errAssetRepo struct{}

func (m *errAssetRepo) List(_ context.Context, _ string) ([]wealth.Asset, error) {
	return nil, errDB
}
func (m *errAssetRepo) Create(_ context.Context, _ wealth.Asset) (wealth.Asset, error) {
	return wealth.Asset{}, errDB
}
func (m *errAssetRepo) Update(_ context.Context, _ wealth.Asset) (wealth.Asset, error) {
	return wealth.Asset{}, errDB
}
func (m *errAssetRepo) Delete(_ context.Context, _, _ string) error { return errDB }

func TestAssetList_DBError(t *testing.T) {
	devEnv(t)
	h := httphandler.NewAssetHandler(&errAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.List))
	req := httptest.NewRequest("GET", "/api/v1/assets", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestAssetDelete_DBError(t *testing.T) {
	devEnv(t)
	h := httphandler.NewAssetHandler(&errAssetRepo{})
	r := chi.NewRouter()
	r.Delete("/assets/{id}", middleware.Auth(http.HandlerFunc(h.Delete)).(http.HandlerFunc))
	req := httptest.NewRequest("DELETE", "/assets/a-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestAssetUpdate_DBError(t *testing.T) {
	devEnv(t)
	h := httphandler.NewAssetHandler(&errAssetRepo{})
	r := chi.NewRouter()
	r.Put("/assets/{id}", middleware.Auth(http.HandlerFunc(h.Update)).(http.HandlerFunc))
	body, _ := json.Marshal(map[string]any{"name": "X", "category": "vehicle"})
	req := httptest.NewRequest("PUT", "/assets/a-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestAssetCreate_DBError(t *testing.T) {
	devEnv(t)
	h := httphandler.NewAssetHandler(&errAssetRepo{})
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

// ── Goal error repos ─────────────────────────────────────────────────────────

type errGoalRepo struct{}

func (m *errGoalRepo) List(_ context.Context, _ string) ([]goals.Goal, error) {
	return nil, errDB
}
func (m *errGoalRepo) Create(_ context.Context, _ goals.Goal) (goals.Goal, error) {
	return goals.Goal{}, errDB
}
func (m *errGoalRepo) Update(_ context.Context, _ goals.Goal) (goals.Goal, error) {
	return goals.Goal{}, errDB
}
func (m *errGoalRepo) Delete(_ context.Context, _, _ string) error { return errDB }
func (m *errGoalRepo) AddKeyResult(_ context.Context, _ goals.KeyResult) (goals.KeyResult, error) {
	return goals.KeyResult{}, errDB
}
func (m *errGoalRepo) UpdateKeyResult(_ context.Context, _ goals.KeyResult) (goals.KeyResult, error) {
	return goals.KeyResult{}, errDB
}
func (m *errGoalRepo) DeleteKeyResult(_ context.Context, _, _ string) error { return errDB }
func (m *errGoalRepo) HabitsSummary(_ context.Context, _ string) (int, int, error) {
	return 0, 0, errDB
}
func (m *errGoalRepo) GoalsAvgProgress(_ context.Context, _ string) (int, error) { return 0, errDB }

func TestGoalList_DBError(t *testing.T) {
	devEnv(t)
	h := httphandler.NewGoalHandler(&errGoalRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.List))
	req := httptest.NewRequest("GET", "/api/v1/goals", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestGoalDelete_DBError(t *testing.T) {
	devEnv(t)
	h := httphandler.NewGoalHandler(&errGoalRepo{})
	r := chi.NewRouter()
	r.Delete("/goals/{id}", middleware.Auth(http.HandlerFunc(h.Delete)).(http.HandlerFunc))
	req := httptest.NewRequest("DELETE", "/goals/g-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestGoalCreate_DBError(t *testing.T) {
	devEnv(t)
	h := httphandler.NewGoalHandler(&errGoalRepo{})
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

func TestGoalUpdate_BadRequest(t *testing.T) {
	devEnv(t)
	h := httphandler.NewGoalHandler(&mockGoalRepo{})
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
	h := httphandler.NewGoalHandler(&errGoalRepo{})
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

func TestGoalAddKeyResult_BadRequest(t *testing.T) {
	devEnv(t)
	h := httphandler.NewGoalHandler(&mockGoalRepo{})
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
	h := httphandler.NewGoalHandler(&errGoalRepo{})
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

func TestGoalUpdateKeyResult_BadRequest(t *testing.T) {
	devEnv(t)
	h := httphandler.NewGoalHandler(&mockGoalRepo{})
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
	h := httphandler.NewGoalHandler(&errGoalRepo{})
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

// ── Notes error repos ────────────────────────────────────────────────────────

type errNoteRepo struct{}

func (m *errNoteRepo) List(_ context.Context, _, _, _ string, _ *bool) ([]notesdomain.Note, error) {
	return nil, errDB
}
func (m *errNoteRepo) Create(_ context.Context, _ notesdomain.Note) (notesdomain.Note, error) {
	return notesdomain.Note{}, errDB
}
func (m *errNoteRepo) Get(_ context.Context, _, _ string) (notesdomain.Note, error) {
	return notesdomain.Note{}, errDB
}
func (m *errNoteRepo) Update(_ context.Context, _ notesdomain.Note) (notesdomain.Note, error) {
	return notesdomain.Note{}, errDB
}
func (m *errNoteRepo) Delete(_ context.Context, _, _ string) error { return errDB }

func TestNoteList_DBError(t *testing.T) {
	devEnv(t)
	h := httphandler.NewNoteHandler(&errNoteRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.List))
	req := httptest.NewRequest("GET", "/api/v1/notes", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestNoteCreate_DBError(t *testing.T) {
	devEnv(t)
	h := httphandler.NewNoteHandler(&errNoteRepo{})
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

func TestNoteDelete_DBError(t *testing.T) {
	devEnv(t)
	h := httphandler.NewNoteHandler(&errNoteRepo{})
	r := chi.NewRouter()
	r.Delete("/notes/{id}", middleware.Auth(http.HandlerFunc(h.Delete)).(http.HandlerFunc))
	req := httptest.NewRequest("DELETE", "/notes/note-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// errNoteUpdateRepo: Get succeeds but Update fails
type errNoteUpdateRepo struct{ errNoteRepo }

func (m *errNoteUpdateRepo) Get(_ context.Context, id, _ string) (notesdomain.Note, error) {
	return notesdomain.Note{ID: id}, nil
}

func TestNoteUpdate_DBError(t *testing.T) {
	devEnv(t)
	h := httphandler.NewNoteHandler(&errNoteUpdateRepo{})
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

// ── Settings error repos ─────────────────────────────────────────────────────

type errSettingsRepo struct{}

func (m *errSettingsRepo) Get(_ context.Context, _ string) (settingsdomain.UserSettings, error) {
	return settingsdomain.UserSettings{}, errDB
}
func (m *errSettingsRepo) Upsert(_ context.Context, _ settingsdomain.UserSettings) (settingsdomain.UserSettings, error) {
	return settingsdomain.UserSettings{}, errDB
}

func TestSettingsGet_DBError(t *testing.T) {
	devEnv(t)
	h := httphandler.NewSettingsHandler(&errSettingsRepo{})
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
	h := httphandler.NewSettingsHandler(&errSettingsRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Update))
	body, _ := json.Marshal(settingsdomain.UserSettings{})
	req := httptest.NewRequest("PUT", "/api/v1/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Transaction error repos ──────────────────────────────────────────────────

type errTxRepo struct{}

func (m *errTxRepo) ListBudgets(_ context.Context, _ string) ([]finance.Budget, error) {
	return nil, errDB
}
func (m *errTxRepo) UpsertBudget(_ context.Context, _ finance.Budget) (finance.Budget, error) {
	return finance.Budget{}, errDB
}
func (m *errTxRepo) DeleteBudget(_ context.Context, _, _ string) error { return errDB }
func (m *errTxRepo) SumByUser(_ context.Context, _ string) (float64, error) { return 0, errDB }
func (m *errTxRepo) SumSpentThisMonth(_ context.Context, _ string) (float64, error) {
	return 0, errDB
}

func TestTransactionListBudgets_DBError(t *testing.T) {
	devEnv(t)
	h := httphandler.NewTransactionHandler(&errTxRepo{})
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
	h := httphandler.NewTransactionHandler(&errTxRepo{})
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

// ── Event error repos ────────────────────────────────────────────────────────

type errEventRepo struct{}

func (m *errEventRepo) List(_ context.Context, _, _, _ string) ([]calendar.Event, error) {
	return nil, errDB
}
func (m *errEventRepo) Create(_ context.Context, _ calendar.Event) (calendar.Event, error) {
	return calendar.Event{}, errDB
}
func (m *errEventRepo) Update(_ context.Context, _ calendar.Event) (calendar.Event, error) {
	return calendar.Event{}, errDB
}
func (m *errEventRepo) Delete(_ context.Context, _, _ string) error { return errDB }
func (m *errEventRepo) UpsertFromGoogle(_ context.Context, _ string, _ []calendar.Event) (int, error) {
	return 0, errDB
}

func TestEventList_DBError(t *testing.T) {
	devEnv(t)
	h := httphandler.NewEventHandler(&errEventRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.List))
	req := httptest.NewRequest("GET", "/api/v1/events", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestEventCreate_DBError(t *testing.T) {
	devEnv(t)
	h := httphandler.NewEventHandler(&errEventRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))
	body, _ := json.Marshal(map[string]any{"title": "X", "start_at": "2026-06-13T09:00:00Z", "end_at": "2026-06-13T10:00:00Z"})
	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestEventDelete_DBError(t *testing.T) {
	devEnv(t)
	h := httphandler.NewEventHandler(&errEventRepo{})
	r := chi.NewRouter()
	r.Delete("/events/{id}", middleware.Auth(http.HandlerFunc(h.Delete)).(http.HandlerFunc))
	req := httptest.NewRequest("DELETE", "/events/evt-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestEventUpdate_DBError(t *testing.T) {
	devEnv(t)
	h := httphandler.NewEventHandler(&errEventRepo{})
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

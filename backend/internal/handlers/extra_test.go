package handlers_test

// Additional tests to cover Update/Delete paths and budget endpoints.
// Uses the mocks already defined in their respective _test.go files.

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/chiutuanbinh/mylifeos/backend/internal/handlers"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"context"
)

// ── Events Update/Delete ─────────────────────────────────────────────────────

func TestEventUpdate(t *testing.T) {
	devEnv(t)
	h := handlers.NewEventHandler(&mockEventRepo{})

	r := chi.NewRouter()
	r.Put("/events/{id}", middleware.Auth(http.HandlerFunc(h.Update)).(http.HandlerFunc))

	body, _ := json.Marshal(map[string]any{"title": "Updated", "start_at": "2026-06-13T09:00:00Z", "end_at": "2026-06-13T10:00:00Z"})
	req := httptest.NewRequest("PUT", "/events/evt-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestEventUpdate_BadRequest(t *testing.T) {
	devEnv(t)
	h := handlers.NewEventHandler(&mockEventRepo{})

	r := chi.NewRouter()
	r.Put("/events/{id}", middleware.Auth(http.HandlerFunc(h.Update)).(http.HandlerFunc))

	req := httptest.NewRequest("PUT", "/events/evt-1", bytes.NewReader([]byte("notjson")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestEventDelete(t *testing.T) {
	devEnv(t)
	h := handlers.NewEventHandler(&mockEventRepo{})

	r := chi.NewRouter()
	r.Delete("/events/{id}", middleware.Auth(http.HandlerFunc(h.Delete)).(http.HandlerFunc))

	req := httptest.NewRequest("DELETE", "/events/evt-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

// ── Notes Update/Delete ──────────────────────────────────────────────────────

func TestNoteUpdate(t *testing.T) {
	devEnv(t)
	h := handlers.NewNoteHandler(&mockNoteRepo{})

	r := chi.NewRouter()
	r.Put("/notes/{id}", middleware.Auth(http.HandlerFunc(h.Update)).(http.HandlerFunc))

	body, _ := json.Marshal(map[string]any{"title": "Updated Note", "content": "new content"})
	req := httptest.NewRequest("PUT", "/notes/note-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestNoteUpdate_BadRequest(t *testing.T) {
	devEnv(t)
	h := handlers.NewNoteHandler(&mockNoteRepo{})

	r := chi.NewRouter()
	r.Put("/notes/{id}", middleware.Auth(http.HandlerFunc(h.Update)).(http.HandlerFunc))

	req := httptest.NewRequest("PUT", "/notes/note-1", bytes.NewReader([]byte("notjson")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestNoteDelete(t *testing.T) {
	devEnv(t)
	h := handlers.NewNoteHandler(&mockNoteRepo{})

	r := chi.NewRouter()
	r.Delete("/notes/{id}", middleware.Auth(http.HandlerFunc(h.Delete)).(http.HandlerFunc))

	req := httptest.NewRequest("DELETE", "/notes/note-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

// ── Transactions ListBudgets/UpsertBudget ────────────────────────────────────

func TestTransactionListBudgets(t *testing.T) {
	devEnv(t)
	h := handlers.NewTransactionHandler(&mockTxRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListBudgets))

	req := httptest.NewRequest("GET", "/api/v1/budgets", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var budgets []models.Budget
	if err := json.NewDecoder(w.Body).Decode(&budgets); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(budgets) == 0 {
		t.Fatal("expected at least one budget")
	}
}

func TestTransactionUpsertBudget(t *testing.T) {
	devEnv(t)
	h := handlers.NewTransactionHandler(&mockTxRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.UpsertBudget))

	body, _ := json.Marshal(map[string]any{"category": "food", "monthly_limit": 600.0})
	req := httptest.NewRequest("PUT", "/api/v1/budgets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTransactionUpsertBudget_BadRequest(t *testing.T) {
	devEnv(t)
	h := handlers.NewTransactionHandler(&mockTxRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.UpsertBudget))

	req := httptest.NewRequest("PUT", "/api/v1/budgets", bytes.NewReader([]byte("notjson")))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ── Error path mocks ─────────────────────────────────────────────────────────

type errEventRepo struct{}

func (m *errEventRepo) List(_ context.Context, _, _, _ string) ([]models.Event, error) {
	return nil, errors.New("db error")
}
func (m *errEventRepo) Create(_ context.Context, _ models.Event) (models.Event, error) {
	return models.Event{}, errors.New("db error")
}
func (m *errEventRepo) Update(_ context.Context, _ models.Event) (models.Event, error) {
	return models.Event{}, errors.New("db error")
}
func (m *errEventRepo) Delete(_ context.Context, _, _ string) error { return errors.New("db error") }
func (m *errEventRepo) UpsertFromGoogle(_ context.Context, _ string, _ []models.Event) (int, error) {
	return 0, errors.New("db error")
}

func TestEventList_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewEventHandler(&errEventRepo{})
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
	h := handlers.NewEventHandler(&errEventRepo{})
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
	h := handlers.NewEventHandler(&errEventRepo{})

	r := chi.NewRouter()
	r.Delete("/events/{id}", middleware.Auth(http.HandlerFunc(h.Delete)).(http.HandlerFunc))

	req := httptest.NewRequest("DELETE", "/events/evt-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

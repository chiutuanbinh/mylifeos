package handlers_test

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
	"github.com/chiutuanbinh/mylifeos/backend/internal/handlers"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
)

var errDB = errors.New("db error")

// ── Asset error repos ────────────────────────────────────────────────────────

type errAssetRepo struct{}

func (m *errAssetRepo) List(_ context.Context, _ string) ([]models.Asset, error) {
	return nil, errDB
}
func (m *errAssetRepo) Create(_ context.Context, _ models.Asset) (models.Asset, error) {
	return models.Asset{}, errDB
}
func (m *errAssetRepo) Update(_ context.Context, _ models.Asset) (models.Asset, error) {
	return models.Asset{}, errDB
}
func (m *errAssetRepo) Delete(_ context.Context, _, _ string) error { return errDB }

func TestAssetList_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewAssetHandler(&errAssetRepo{})
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
	h := handlers.NewAssetHandler(&errAssetRepo{})
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
	h := handlers.NewAssetHandler(&errAssetRepo{})
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

// ── Goal error repos ─────────────────────────────────────────────────────────

type errGoalRepo struct{}

func (m *errGoalRepo) List(_ context.Context, _ string) ([]models.Goal, error) {
	return nil, errDB
}
func (m *errGoalRepo) Create(_ context.Context, _ models.Goal) (models.Goal, error) {
	return models.Goal{}, errDB
}
func (m *errGoalRepo) Update(_ context.Context, _ models.Goal) (models.Goal, error) {
	return models.Goal{}, errDB
}
func (m *errGoalRepo) Delete(_ context.Context, _, _ string) error { return errDB }
func (m *errGoalRepo) AddKeyResult(_ context.Context, _ models.KeyResult) (models.KeyResult, error) {
	return models.KeyResult{}, errDB
}
func (m *errGoalRepo) UpdateKeyResult(_ context.Context, _ models.KeyResult) (models.KeyResult, error) {
	return models.KeyResult{}, errDB
}
func (m *errGoalRepo) DeleteKeyResult(_ context.Context, _, _ string) error { return errDB }

func TestGoalList_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewGoalHandler(&errGoalRepo{})
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
	h := handlers.NewGoalHandler(&errGoalRepo{})
	r := chi.NewRouter()
	r.Delete("/goals/{id}", middleware.Auth(http.HandlerFunc(h.Delete)).(http.HandlerFunc))
	req := httptest.NewRequest("DELETE", "/goals/g-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Notes error repos ────────────────────────────────────────────────────────

type errNoteRepo struct{}

func (m *errNoteRepo) List(_ context.Context, _, _, _ string, _ *bool) ([]models.Note, error) {
	return nil, errDB
}
func (m *errNoteRepo) Create(_ context.Context, _ models.Note) (models.Note, error) {
	return models.Note{}, errDB
}
func (m *errNoteRepo) Get(_ context.Context, _, _ string) (models.Note, error) {
	return models.Note{}, errDB
}
func (m *errNoteRepo) Update(_ context.Context, _ models.Note) (models.Note, error) {
	return models.Note{}, errDB
}
func (m *errNoteRepo) Delete(_ context.Context, _, _ string) error { return errDB }

func TestNoteList_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewNoteHandler(&errNoteRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.List))
	req := httptest.NewRequest("GET", "/api/v1/notes", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestNoteDelete_DBError(t *testing.T) {
	devEnv(t)
	h := handlers.NewNoteHandler(&errNoteRepo{})
	r := chi.NewRouter()
	r.Delete("/notes/{id}", middleware.Auth(http.HandlerFunc(h.Delete)).(http.HandlerFunc))
	req := httptest.NewRequest("DELETE", "/notes/note-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/handlers"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
)

type mockNoteRepo struct{ created *models.Note }

func (m *mockNoteRepo) List(_ context.Context, _, _, _ string, _ *bool) ([]models.Note, error) {
	return []models.Note{{ID: "note-1", Title: "Test Note", Content: "hello"}}, nil
}
func (m *mockNoteRepo) Create(_ context.Context, n models.Note) (models.Note, error) {
	n.ID = "note-new"
	m.created = &n
	return n, nil
}
func (m *mockNoteRepo) Update(_ context.Context, n models.Note) (models.Note, error) { return n, nil }
func (m *mockNoteRepo) Delete(_ context.Context, _, _ string) error                  { return nil }

func TestNoteList(t *testing.T) {
	devEnv(t)
	h := handlers.NewNoteHandler(&mockNoteRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.List))

	req := httptest.NewRequest("GET", "/api/v1/notes", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var notes []models.Note
	if err := json.NewDecoder(w.Body).Decode(&notes); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(notes) != 1 || notes[0].Title != "Test Note" {
		t.Errorf("unexpected notes: %+v", notes)
	}
}

func TestNoteCreate(t *testing.T) {
	devEnv(t)
	mock := &mockNoteRepo{}
	h := handlers.NewNoteHandler(mock)
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(models.Note{Title: "My Note", Content: "content here"})
	req := httptest.NewRequest("POST", "/api/v1/notes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if mock.created == nil {
		t.Fatal("repo.Create not called")
	}
	if mock.created.UserID != "user-1" {
		t.Errorf("expected user_id=user-1, got %s", mock.created.UserID)
	}
}

func TestNoteCreateBadBody(t *testing.T) {
	devEnv(t)
	h := handlers.NewNoteHandler(&mockNoteRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	req := httptest.NewRequest("POST", "/api/v1/notes", bytes.NewBufferString("{bad json"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

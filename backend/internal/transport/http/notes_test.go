package httphandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	httphandler "github.com/chiutuanbinh/mylifeos/backend/internal/transport/http"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	notesdomain "github.com/chiutuanbinh/mylifeos/backend/internal/domain/notes"
)

type mockNoteRepo struct{ created *notesdomain.Note }

func (m *mockNoteRepo) List(_ context.Context, _, _, _ string, _ *bool) ([]notesdomain.Note, error) {
	return []notesdomain.Note{{ID: "note-1", Title: "Test Note", Content: "hello"}}, nil
}
func (m *mockNoteRepo) Create(_ context.Context, n notesdomain.Note) (notesdomain.Note, error) {
	n.ID = "note-new"
	m.created = &n
	return n, nil
}
func (m *mockNoteRepo) Get(_ context.Context, id, _ string) (notesdomain.Note, error) {
	return notesdomain.Note{ID: id, Title: "Existing Title", Content: "Existing Content", Tags: []string{"existing"}, Pinned: false}, nil
}
func (m *mockNoteRepo) Update(_ context.Context, n notesdomain.Note) (notesdomain.Note, error) {
	return n, nil
}
func (m *mockNoteRepo) Delete(_ context.Context, _, _ string) error { return nil }

func TestNoteList(t *testing.T) {
	devEnv(t)
	h := httphandler.NewNoteHandler(&mockNoteRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.List))

	req := httptest.NewRequest("GET", "/api/v1/notes", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var notes []notesdomain.Note
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
	h := httphandler.NewNoteHandler(mock)
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(notesdomain.Note{Title: "My Note", Content: "content here"})
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
	h := httphandler.NewNoteHandler(&mockNoteRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	req := httptest.NewRequest("POST", "/api/v1/notes", bytes.NewBufferString("{bad json"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestNoteUpdate(t *testing.T) {
	devEnv(t)
	h := httphandler.NewNoteHandler(&mockNoteRepo{})

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
	h := httphandler.NewNoteHandler(&mockNoteRepo{})

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
	h := httphandler.NewNoteHandler(&mockNoteRepo{})

	r := chi.NewRouter()
	r.Delete("/notes/{id}", middleware.Auth(http.HandlerFunc(h.Delete)).(http.HandlerFunc))

	req := httptest.NewRequest("DELETE", "/notes/note-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestNoteList_PinnedFalse(t *testing.T) {
	devEnv(t)
	h := httphandler.NewNoteHandler(&mockNoteRepo{})
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
	h := httphandler.NewNoteHandler(&mockNoteRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.List))
	req := httptest.NewRequest("GET", "/api/v1/notes?pinned=true", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

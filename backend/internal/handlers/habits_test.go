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

type mockHabitRepo struct{}

func (m *mockHabitRepo) List(_ context.Context, _ string) ([]models.Habit, error) {
	return []models.Habit{{ID: "h-1", Name: "Exercise", Icon: "🏃"}}, nil
}
func (m *mockHabitRepo) Create(_ context.Context, h models.Habit) (models.Habit, error) {
	h.ID = "h-new"
	return h, nil
}
func (m *mockHabitRepo) Delete(_ context.Context, _, _ string) error { return nil }
func (m *mockHabitRepo) GetLogs(_ context.Context, _, _ string) ([]models.HabitLog, error) {
	return []models.HabitLog{{ID: "hl-1", HabitID: "h-1", Done: true}}, nil
}
func (m *mockHabitRepo) ToggleLog(_ context.Context, _, _, _ string) (models.HabitLog, error) {
	return models.HabitLog{ID: "hl-1", Done: true}, nil
}

func TestHabitList(t *testing.T) {
	devEnv(t)
	h := handlers.NewHabitHandler(&mockHabitRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.List))

	req := httptest.NewRequest("GET", "/api/v1/habits", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var habits []models.Habit
	if err := json.NewDecoder(w.Body).Decode(&habits); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(habits) != 1 || habits[0].ID != "h-1" {
		t.Fatalf("unexpected: %+v", habits)
	}
}

func TestHabitCreate(t *testing.T) {
	devEnv(t)
	h := handlers.NewHabitHandler(&mockHabitRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(map[string]any{"name": "Meditate", "icon": "🧘"})
	req := httptest.NewRequest("POST", "/api/v1/habits", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHabitCreate_DefaultIcon(t *testing.T) {
	devEnv(t)
	h := handlers.NewHabitHandler(&mockHabitRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(map[string]any{"name": "No icon habit"})
	req := httptest.NewRequest("POST", "/api/v1/habits", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var habit models.Habit
	json.NewDecoder(w.Body).Decode(&habit)
	if habit.Icon != "✓" {
		t.Fatalf("expected default icon '✓', got %q", habit.Icon)
	}
}

func TestHabitCreate_BadRequest(t *testing.T) {
	devEnv(t)
	h := handlers.NewHabitHandler(&mockHabitRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	req := httptest.NewRequest("POST", "/api/v1/habits", bytes.NewReader([]byte("notjson")))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHabitDelete(t *testing.T) {
	devEnv(t)
	h := handlers.NewHabitHandler(&mockHabitRepo{})

	r := chi.NewRouter()
	r.Delete("/habits/{id}", middleware.Auth(http.HandlerFunc(h.Delete)).(http.HandlerFunc))

	req := httptest.NewRequest("DELETE", "/habits/h-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestHabitGetLogs(t *testing.T) {
	devEnv(t)
	h := handlers.NewHabitHandler(&mockHabitRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.GetLogs))

	req := httptest.NewRequest("GET", "/api/v1/habits/logs?date=2026-06-13", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHabitToggleLog(t *testing.T) {
	devEnv(t)
	h := handlers.NewHabitHandler(&mockHabitRepo{})

	r := chi.NewRouter()
	r.Post("/habits/{id}/toggle", middleware.Auth(http.HandlerFunc(h.ToggleLog)).(http.HandlerFunc))

	body, _ := json.Marshal(map[string]any{"date": "2026-06-13"})
	req := httptest.NewRequest("POST", "/habits/h-1/toggle", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

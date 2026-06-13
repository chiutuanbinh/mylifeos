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

type mockGoalRepo struct{}

func (m *mockGoalRepo) List(_ context.Context, _ string) ([]models.Goal, error) {
	return []models.Goal{{ID: "g-1", Name: "Fitness", Color: "#1677ff", Status: "active"}}, nil
}
func (m *mockGoalRepo) Create(_ context.Context, g models.Goal) (models.Goal, error) {
	g.ID = "g-new"
	return g, nil
}
func (m *mockGoalRepo) Update(_ context.Context, g models.Goal) (models.Goal, error) { return g, nil }
func (m *mockGoalRepo) Delete(_ context.Context, _, _ string) error                  { return nil }
func (m *mockGoalRepo) AddKeyResult(_ context.Context, kr models.KeyResult) (models.KeyResult, error) {
	kr.ID = "kr-new"
	return kr, nil
}
func (m *mockGoalRepo) UpdateKeyResult(_ context.Context, kr models.KeyResult) (models.KeyResult, error) {
	return kr, nil
}
func (m *mockGoalRepo) DeleteKeyResult(_ context.Context, _, _ string) error { return nil }

func TestGoalList(t *testing.T) {
	devEnv(t)
	h := handlers.NewGoalHandler(&mockGoalRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.List))

	req := httptest.NewRequest("GET", "/api/v1/goals", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var goals []models.Goal
	if err := json.NewDecoder(w.Body).Decode(&goals); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(goals) != 1 || goals[0].ID != "g-1" {
		t.Fatalf("unexpected: %+v", goals)
	}
}

func TestGoalCreate(t *testing.T) {
	devEnv(t)
	h := handlers.NewGoalHandler(&mockGoalRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(map[string]any{"name": "Learn Go", "color": "#ff0000"})
	req := httptest.NewRequest("POST", "/api/v1/goals", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGoalCreate_DefaultColor(t *testing.T) {
	devEnv(t)
	h := handlers.NewGoalHandler(&mockGoalRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(map[string]any{"name": "No color goal"})
	req := httptest.NewRequest("POST", "/api/v1/goals", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var g models.Goal
	json.NewDecoder(w.Body).Decode(&g)
	if g.Color != "#1677ff" {
		t.Fatalf("expected default color, got %s", g.Color)
	}
}

func TestGoalCreate_BadRequest(t *testing.T) {
	devEnv(t)
	h := handlers.NewGoalHandler(&mockGoalRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	req := httptest.NewRequest("POST", "/api/v1/goals", bytes.NewReader([]byte("notjson")))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGoalUpdate(t *testing.T) {
	devEnv(t)
	h := handlers.NewGoalHandler(&mockGoalRepo{})

	r := chi.NewRouter()
	r.Put("/goals/{id}", middleware.Auth(http.HandlerFunc(h.Update)).(http.HandlerFunc))

	body, _ := json.Marshal(map[string]any{"name": "Updated Goal"})
	req := httptest.NewRequest("PUT", "/goals/g-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGoalDelete(t *testing.T) {
	devEnv(t)
	h := handlers.NewGoalHandler(&mockGoalRepo{})

	r := chi.NewRouter()
	r.Delete("/goals/{id}", middleware.Auth(http.HandlerFunc(h.Delete)).(http.HandlerFunc))

	req := httptest.NewRequest("DELETE", "/goals/g-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestGoalAddKeyResult(t *testing.T) {
	devEnv(t)
	h := handlers.NewGoalHandler(&mockGoalRepo{})

	r := chi.NewRouter()
	r.Post("/goals/{id}/key-results", middleware.Auth(http.HandlerFunc(h.AddKeyResult)).(http.HandlerFunc))

	body, _ := json.Marshal(map[string]any{"description": "Run 5k"})
	req := httptest.NewRequest("POST", "/goals/g-1/key-results", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGoalUpdateKeyResult(t *testing.T) {
	devEnv(t)
	h := handlers.NewGoalHandler(&mockGoalRepo{})

	r := chi.NewRouter()
	r.Put("/goals/{id}/key-results/{kr_id}", middleware.Auth(http.HandlerFunc(h.UpdateKeyResult)).(http.HandlerFunc))

	body, _ := json.Marshal(map[string]any{"done": true})
	req := httptest.NewRequest("PUT", "/goals/g-1/key-results/kr-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGoalDeleteKeyResult(t *testing.T) {
	devEnv(t)
	h := handlers.NewGoalHandler(&mockGoalRepo{})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "g-1")
	rctx.URLParams.Add("kr_id", "kr-1")

	req := httptest.NewRequest("DELETE", "/api/v1/goals/g-1/key-results/kr-1", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := middleware.Auth(http.HandlerFunc(h.DeleteKeyResult))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

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

type mockAssetRepo struct{}

func (m *mockAssetRepo) List(_ context.Context, _ string) ([]models.Asset, error) {
	return []models.Asset{{ID: "a-1", Name: "Car", Category: "vehicle", Value: 10000}}, nil
}
func (m *mockAssetRepo) Create(_ context.Context, a models.Asset) (models.Asset, error) {
	a.ID = "a-new"
	return a, nil
}
func (m *mockAssetRepo) Update(_ context.Context, a models.Asset) (models.Asset, error) { return a, nil }
func (m *mockAssetRepo) Delete(_ context.Context, _, _ string) error                    { return nil }

func TestAssetList(t *testing.T) {
	devEnv(t)
	h := handlers.NewAssetHandler(&mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.List))

	req := httptest.NewRequest("GET", "/api/v1/assets", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var assets []models.Asset
	if err := json.NewDecoder(w.Body).Decode(&assets); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(assets) != 1 || assets[0].ID != "a-1" {
		t.Fatalf("unexpected: %+v", assets)
	}
}

func TestAssetCreate(t *testing.T) {
	devEnv(t)
	h := handlers.NewAssetHandler(&mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(map[string]any{"name": "Laptop", "category": "electronics", "value": 1500.0})
	req := httptest.NewRequest("POST", "/api/v1/assets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAssetCreate_BadRequest(t *testing.T) {
	devEnv(t)
	h := handlers.NewAssetHandler(&mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	req := httptest.NewRequest("POST", "/api/v1/assets", bytes.NewReader([]byte("notjson")))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAssetUpdate(t *testing.T) {
	devEnv(t)
	h := handlers.NewAssetHandler(&mockAssetRepo{})

	r := chi.NewRouter()
	r.Put("/assets/{id}", middleware.Auth(http.HandlerFunc(h.Update)).(http.HandlerFunc))

	body, _ := json.Marshal(map[string]any{"name": "Updated", "value": 2000.0})
	req := httptest.NewRequest("PUT", "/assets/a-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAssetDelete(t *testing.T) {
	devEnv(t)
	h := handlers.NewAssetHandler(&mockAssetRepo{})

	r := chi.NewRouter()
	r.Delete("/assets/{id}", middleware.Auth(http.HandlerFunc(h.Delete)).(http.HandlerFunc))

	req := httptest.NewRequest("DELETE", "/assets/a-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

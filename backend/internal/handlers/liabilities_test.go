package handlers_test

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

var errLiabilityInternal = errors.New("db error")

type mockLiabilityRepo struct{ failNext bool }

type mockLiabilityRepoErr struct{}

func (m *mockLiabilityRepoErr) List(_ context.Context, _ string) ([]models.Liability, error) {
	return nil, errLiabilityInternal
}
func (m *mockLiabilityRepoErr) Create(_ context.Context, l models.Liability) (models.Liability, error) {
	return l, errLiabilityInternal
}
func (m *mockLiabilityRepoErr) Update(_ context.Context, l models.Liability) (models.Liability, error) {
	return l, errLiabilityInternal
}
func (m *mockLiabilityRepoErr) Delete(_ context.Context, _, _ string) error { return errLiabilityInternal }
func (m *mockLiabilityRepoErr) TotalBalance(_ context.Context, _ string) (float64, error) {
	return 0, errLiabilityInternal
}

func (m *mockLiabilityRepo) List(_ context.Context, _ string) ([]models.Liability, error) {
	ir := 0.085
	return []models.Liability{{ID: "l-1", Name: "Car Loan", Category: "Car Loan", Balance: 200000000, InterestRate: &ir}}, nil
}
func (m *mockLiabilityRepo) Create(_ context.Context, l models.Liability) (models.Liability, error) {
	l.ID = "l-new"
	return l, nil
}
func (m *mockLiabilityRepo) Update(_ context.Context, l models.Liability) (models.Liability, error) {
	return l, nil
}
func (m *mockLiabilityRepo) Delete(_ context.Context, _, _ string) error { return nil }
func (m *mockLiabilityRepo) TotalBalance(_ context.Context, _ string) (float64, error) {
	return 200000000, nil
}

func TestLiabilityList(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.List))

	req := httptest.NewRequest("GET", "/api/v1/liabilities", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var items []models.Liability
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) != 1 || items[0].ID != "l-1" {
		t.Fatalf("unexpected: %+v", items)
	}
}

func TestLiabilityCreate(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(map[string]any{"name": "Mortgage", "category": "Mortgage", "balance": 500000000.0})
	req := httptest.NewRequest("POST", "/api/v1/liabilities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLiabilityCreate_MissingName(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(map[string]any{"category": "Mortgage", "balance": 500000000.0})
	req := httptest.NewRequest("POST", "/api/v1/liabilities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLiabilityCreate_NegativeBalance(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(map[string]any{"name": "Loan", "category": "Personal Loan", "balance": -1000.0})
	req := httptest.NewRequest("POST", "/api/v1/liabilities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLiabilityCreate_InvalidInterestRate(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(map[string]any{"name": "Loan", "category": "Personal Loan", "balance": 1000.0, "interest_rate": 1.5})
	req := httptest.NewRequest("POST", "/api/v1/liabilities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLiabilityCreate_BadJSON(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	req := httptest.NewRequest("POST", "/api/v1/liabilities", bytes.NewReader([]byte("notjson")))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLiabilityUpdate(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepo{})
	r := chi.NewRouter()
	r.Patch("/liabilities/{id}", middleware.Auth(http.HandlerFunc(h.Update)).(http.HandlerFunc))

	body, _ := json.Marshal(map[string]any{"name": "Updated Loan", "category": "Car Loan", "balance": 180000000.0})
	req := httptest.NewRequest("PATCH", "/liabilities/l-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestLiabilityDelete(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepo{})
	r := chi.NewRouter()
	r.Delete("/liabilities/{id}", middleware.Auth(http.HandlerFunc(h.Delete)).(http.HandlerFunc))

	req := httptest.NewRequest("DELETE", "/liabilities/l-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestLiabilityCreate_MissingCategory(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(map[string]any{"name": "Loan", "balance": 1000.0})
	req := httptest.NewRequest("POST", "/api/v1/liabilities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLiabilityList_RepoError(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepoErr{})
	handler := middleware.Auth(http.HandlerFunc(h.List))

	req := httptest.NewRequest("GET", "/api/v1/liabilities", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestLiabilityCreate_RepoError(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepoErr{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(map[string]any{"name": "Loan", "category": "Personal Loan", "balance": 1000.0})
	req := httptest.NewRequest("POST", "/api/v1/liabilities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestLiabilityUpdate_BadJSON(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepo{})
	r := chi.NewRouter()
	r.Patch("/liabilities/{id}", middleware.Auth(http.HandlerFunc(h.Update)).(http.HandlerFunc))

	req := httptest.NewRequest("PATCH", "/liabilities/l-1", bytes.NewReader([]byte("notjson")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLiabilityUpdate_ValidationError(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepo{})
	r := chi.NewRouter()
	r.Patch("/liabilities/{id}", middleware.Auth(http.HandlerFunc(h.Update)).(http.HandlerFunc))

	body, _ := json.Marshal(map[string]any{"name": "", "category": "Car Loan", "balance": 1000.0})
	req := httptest.NewRequest("PATCH", "/liabilities/l-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLiabilityUpdate_RepoError(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepoErr{})
	r := chi.NewRouter()
	r.Patch("/liabilities/{id}", middleware.Auth(http.HandlerFunc(h.Update)).(http.HandlerFunc))

	body, _ := json.Marshal(map[string]any{"name": "Loan", "category": "Car Loan", "balance": 1000.0})
	req := httptest.NewRequest("PATCH", "/liabilities/l-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestLiabilityDelete_RepoError(t *testing.T) {
	devEnv(t)
	h := handlers.NewLiabilityHandler(&mockLiabilityRepoErr{})
	r := chi.NewRouter()
	r.Delete("/liabilities/{id}", middleware.Auth(http.HandlerFunc(h.Delete)).(http.HandlerFunc))

	req := httptest.NewRequest("DELETE", "/liabilities/l-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

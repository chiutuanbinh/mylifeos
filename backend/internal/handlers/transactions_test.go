package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/chiutuanbinh/mylifeos/backend/internal/handlers"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
)

type mockTxRepo struct{ created *models.Transaction }

func (m *mockTxRepo) List(_ context.Context, _, _, _, _ string, _, _ int) ([]models.Transaction, error) {
	return []models.Transaction{{ID: "tx-1", Amount: 50.0, Category: "food"}}, nil
}
func (m *mockTxRepo) Create(_ context.Context, t models.Transaction) (models.Transaction, error) {
	t.ID = "tx-new"
	m.created = &t
	return t, nil
}
func (m *mockTxRepo) Delete(_ context.Context, _, _ string) error { return nil }
func (m *mockTxRepo) ListBudgets(_ context.Context, _ string) ([]models.Budget, error) {
	return []models.Budget{{Category: "food", MonthlyLimit: 500}}, nil
}
func (m *mockTxRepo) UpsertBudget(_ context.Context, b models.Budget) (models.Budget, error) {
	return b, nil
}

func devEnv(t *testing.T) {
	t.Helper()
	os.Setenv("ENV", "development")
	os.Setenv("DEV_USER_ID", "user-1")
	t.Cleanup(func() {
		os.Unsetenv("ENV")
		os.Unsetenv("DEV_USER_ID")
	})
}

func TestTransactionList(t *testing.T) {
	devEnv(t)
	mock := &mockTxRepo{}
	h := handlers.NewTransactionHandler(mock)
	handler := middleware.Auth(http.HandlerFunc(h.List))

	req := httptest.NewRequest("GET", "/api/v1/transactions", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var txs []models.Transaction
	if err := json.NewDecoder(w.Body).Decode(&txs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(txs) != 1 {
		t.Errorf("expected 1 transaction, got %d", len(txs))
	}
}

func TestTransactionCreate(t *testing.T) {
	devEnv(t)
	mock := &mockTxRepo{}
	h := handlers.NewTransactionHandler(mock)
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(models.Transaction{Amount: 25.0, Category: "transport", Description: "bus"})
	req := httptest.NewRequest("POST", "/api/v1/transactions", bytes.NewReader(body))
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

func TestTransactionCreateBadBody(t *testing.T) {
	devEnv(t)
	h := handlers.NewTransactionHandler(&mockTxRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	req := httptest.NewRequest("POST", "/api/v1/transactions", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestTransactionDelete(t *testing.T) {
	devEnv(t)
	h := handlers.NewTransactionHandler(&mockTxRepo{})

	// Use chi router to inject URL param
	r := chi.NewRouter()
	r.Delete("/transactions/{id}", h.Delete)
	router := middleware.Auth(r)

	req := httptest.NewRequest("DELETE", "/transactions/tx-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

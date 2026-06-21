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
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/finance"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
)

type mockTxRepo struct {
	created *finance.Transaction
	budgets []finance.Budget
}

func (m *mockTxRepo) List(_ context.Context, _, _, _, _ string, _, _ int) ([]finance.Transaction, error) {
	return []finance.Transaction{{ID: "tx-1", Amount: 50.0, Category: "food"}}, nil
}
func (m *mockTxRepo) Create(_ context.Context, t finance.Transaction) (finance.Transaction, error) {
	t.ID = "tx-new"
	m.created = &t
	return t, nil
}
func (m *mockTxRepo) Delete(_ context.Context, _, _ string) error { return nil }
func (m *mockTxRepo) ListBudgets(_ context.Context, _ string) ([]finance.Budget, error) {
	return []finance.Budget{{Category: "food", MonthlyLimit: 500}}, nil
}
func (m *mockTxRepo) UpsertBudget(_ context.Context, b finance.Budget) (finance.Budget, error) {
	return b, nil
}
func (m *mockTxRepo) DeleteBudget(_ context.Context, userID, category string) error {
	for i, b := range m.budgets {
		if b.UserID == userID && b.Category == category {
			m.budgets = append(m.budgets[:i], m.budgets[i+1:]...)
			return nil
		}
	}
	return repository.ErrBudgetNotFound
}
func (m *mockTxRepo) SumByUser(_ context.Context, _ string) (float64, error)         { return 0, nil }
func (m *mockTxRepo) SumSpentThisMonth(_ context.Context, _ string) (float64, error) { return 0, nil }

func TestTransactionList(t *testing.T) {
	devEnv(t)
	mock := &mockTxRepo{}
	h := httphandler.NewTransactionHandler(mock)
	handler := middleware.Auth(http.HandlerFunc(h.List))

	req := httptest.NewRequest("GET", "/api/v1/transactions", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var txs []finance.Transaction
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
	h := httphandler.NewTransactionHandler(mock)
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(finance.Transaction{Amount: 25.0, Category: "transport", Description: "bus"})
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
	h := httphandler.NewTransactionHandler(&mockTxRepo{})
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
	h := httphandler.NewTransactionHandler(&mockTxRepo{})

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

func TestTransactionListBudgets(t *testing.T) {
	devEnv(t)
	h := httphandler.NewTransactionHandler(&mockTxRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListBudgets))

	req := httptest.NewRequest("GET", "/api/v1/budgets", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var budgets []finance.Budget
	if err := json.NewDecoder(w.Body).Decode(&budgets); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(budgets) == 0 {
		t.Fatal("expected at least one budget")
	}
}

func TestTransactionUpsertBudget(t *testing.T) {
	devEnv(t)
	h := httphandler.NewTransactionHandler(&mockTxRepo{})
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
	h := httphandler.NewTransactionHandler(&mockTxRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.UpsertBudget))

	req := httptest.NewRequest("PUT", "/api/v1/budgets", bytes.NewReader([]byte("notjson")))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTransactionHandler_DeleteBudget(t *testing.T) {
	repo := &mockTxRepo{
		budgets: []finance.Budget{
			{ID: "b1", UserID: "user-1", Category: "Food", MonthlyLimit: 500000},
		},
	}
	h := httphandler.NewTransactionHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/budgets/Food", nil)
	req = req.WithContext(withUserID(req.Context(), "user-1"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, func() *chi.Context {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("category", "Food")
		return rctx
	}()))
	rr := httptest.NewRecorder()
	h.DeleteBudget(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d: %s", rr.Code, rr.Body.String())
	}
	if len(repo.budgets) != 0 {
		t.Errorf("expected budget to be deleted, still %d budgets", len(repo.budgets))
	}
}

func TestTransactionHandler_DeleteBudget_NotFound(t *testing.T) {
	repo := &mockTxRepo{budgets: []finance.Budget{}}
	h := httphandler.NewTransactionHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/budgets/Nonexistent", nil)
	req = req.WithContext(withUserID(req.Context(), "user-1"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, func() *chi.Context {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("category", "Nonexistent")
		return rctx
	}()))
	rr := httptest.NewRecorder()
	h.DeleteBudget(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rr.Code)
	}
}

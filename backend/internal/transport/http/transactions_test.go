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
	budgets []finance.Budget
}

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

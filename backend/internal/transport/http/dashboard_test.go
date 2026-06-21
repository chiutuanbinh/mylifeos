package httphandler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	httphandler "github.com/chiutuanbinh/mylifeos/backend/internal/transport/http"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/finance"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/goals"
	trendsdomain "github.com/chiutuanbinh/mylifeos/backend/internal/domain/trends"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/wealth"
	dashboardsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/dashboard"
)

// --- stubs for dashboard service ---

type stubAssetRepo struct{}

func (s *stubAssetRepo) List(_ context.Context, _ string) ([]wealth.Asset, error) { return nil, nil }
func (s *stubAssetRepo) Create(_ context.Context, a wealth.Asset) (wealth.Asset, error) {
	return a, nil
}
func (s *stubAssetRepo) Update(_ context.Context, a wealth.Asset) (wealth.Asset, error) {
	return a, nil
}
func (s *stubAssetRepo) Delete(_ context.Context, _, _ string) error { return nil }

type stubLiabilityRepo struct{}

func (s *stubLiabilityRepo) List(_ context.Context, _ string) ([]wealth.Liability, error) {
	return nil, nil
}
func (s *stubLiabilityRepo) Create(_ context.Context, l wealth.Liability) (wealth.Liability, error) {
	return l, nil
}
func (s *stubLiabilityRepo) Update(_ context.Context, l wealth.Liability) (wealth.Liability, error) {
	return l, nil
}
func (s *stubLiabilityRepo) Delete(_ context.Context, _, _ string) error            { return nil }
func (s *stubLiabilityRepo) TotalBalance(_ context.Context, _ string) (float64, error) {
	return 0, nil
}

type stubTxRepo struct{}

func (s *stubTxRepo) List(_ context.Context, _, _, _, _ string, _, _ int) ([]finance.Transaction, error) {
	return []finance.Transaction{}, nil
}
func (s *stubTxRepo) Create(_ context.Context, t finance.Transaction) (finance.Transaction, error) {
	return t, nil
}
func (s *stubTxRepo) Delete(_ context.Context, _, _ string) error { return nil }
func (s *stubTxRepo) ListBudgets(_ context.Context, _ string) ([]finance.Budget, error) {
	return nil, nil
}
func (s *stubTxRepo) UpsertBudget(_ context.Context, b finance.Budget) (finance.Budget, error) {
	return b, nil
}
func (s *stubTxRepo) DeleteBudget(_ context.Context, _, _ string) error               { return nil }
func (s *stubTxRepo) SumByUser(_ context.Context, _ string) (float64, error)         { return 0, nil }
func (s *stubTxRepo) SumSpentThisMonth(_ context.Context, _ string) (float64, error) { return 0, nil }

type stubGoalRepo struct{}

func (s *stubGoalRepo) List(_ context.Context, _ string) ([]goals.Goal, error) { return nil, nil }
func (s *stubGoalRepo) Create(_ context.Context, g goals.Goal) (goals.Goal, error) { return g, nil }
func (s *stubGoalRepo) Update(_ context.Context, g goals.Goal) (goals.Goal, error) { return g, nil }
func (s *stubGoalRepo) Delete(_ context.Context, _, _ string) error                { return nil }
func (s *stubGoalRepo) AddKeyResult(_ context.Context, kr goals.KeyResult) (goals.KeyResult, error) {
	return kr, nil
}
func (s *stubGoalRepo) UpdateKeyResult(_ context.Context, kr goals.KeyResult) (goals.KeyResult, error) {
	return kr, nil
}
func (s *stubGoalRepo) DeleteKeyResult(_ context.Context, _, _ string) error { return nil }
func (s *stubGoalRepo) HabitsSummary(_ context.Context, _ string) (int, int, error) {
	return 5, 3, nil
}
func (s *stubGoalRepo) GoalsAvgProgress(_ context.Context, _ string) (int, error) { return 65, nil }

type stubTrendsRepoForDash struct{}

func (s *stubTrendsRepoForDash) ListSnapshots(_ context.Context, _ string) ([]trendsdomain.NetWorthSnapshot, error) {
	return nil, nil
}
func (s *stubTrendsRepoForDash) UpsertSnapshot(_ context.Context, snap trendsdomain.NetWorthSnapshot) (trendsdomain.NetWorthSnapshot, error) {
	return snap, nil
}
func (s *stubTrendsRepoForDash) UpsertBenchmark(_ context.Context, _ trendsdomain.BenchmarkData) error {
	return nil
}
func (s *stubTrendsRepoForDash) ListBenchmarks(_ context.Context, _ []string, _, _ string) ([]trendsdomain.BenchmarkData, error) {
	return nil, nil
}
func (s *stubTrendsRepoForDash) LatestBankRates(_ context.Context) ([]trendsdomain.BankRate, error) {
	return nil, nil
}
func (s *stubTrendsRepoForDash) UpsertBankRate(_ context.Context, _ trendsdomain.BankRate) error {
	return nil
}
func (s *stubTrendsRepoForDash) ListNews(_ context.Context, _ int) ([]trendsdomain.NewsItem, error) {
	return nil, nil
}
func (s *stubTrendsRepoForDash) UpsertNews(_ context.Context, _ []trendsdomain.NewsItem) error {
	return nil
}

// errAssetRepoForDash returns error so dashboard.Summary returns err.
type errAssetRepoForDash struct{}

func (s *errAssetRepoForDash) List(_ context.Context, _ string) ([]wealth.Asset, error) {
	return nil, errors.New("db error")
}
func (s *errAssetRepoForDash) Create(_ context.Context, a wealth.Asset) (wealth.Asset, error) {
	return a, nil
}
func (s *errAssetRepoForDash) Update(_ context.Context, a wealth.Asset) (wealth.Asset, error) {
	return a, nil
}
func (s *errAssetRepoForDash) Delete(_ context.Context, _, _ string) error { return nil }

func newDashboardHandler() *httphandler.DashboardHandler {
	svc := dashboardsvc.New(
		&stubAssetRepo{},
		&stubLiabilityRepo{},
		&stubTxRepo{},
		&stubGoalRepo{},
		&stubTrendsRepoForDash{},
	)
	return httphandler.NewDashboardHandler(svc)
}

func TestDashboardSummary(t *testing.T) {
	devEnv(t)

	h := newDashboardHandler()
	handler := middleware.Auth(http.HandlerFunc(h.Summary))

	req := httptest.NewRequest("GET", "/api/v1/dashboard/summary", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result dashboardsvc.Summary
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.HabitsTotal != 5 {
		t.Errorf("expected 5 habits, got %d", result.HabitsTotal)
	}
}

func TestDashboardSummary_DBError(t *testing.T) {
	devEnv(t)
	svc := dashboardsvc.New(
		&errAssetRepoForDash{},
		&stubLiabilityRepo{},
		&stubTxRepo{},
		&stubGoalRepo{},
		&stubTrendsRepoForDash{},
	)
	h := httphandler.NewDashboardHandler(svc)
	handler := middleware.Auth(http.HandlerFunc(h.Summary))
	req := httptest.NewRequest("GET", "/api/v1/dashboard/summary", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

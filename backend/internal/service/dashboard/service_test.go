package dashboardsvc

import (
	"context"
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/finance"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/goals"
	trendsdomain "github.com/chiutuanbinh/mylifeos/backend/internal/domain/trends"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/wealth"
)

// --- stubs ---

type stubAssetRepo struct{ assets []wealth.Asset }

func (s *stubAssetRepo) List(_ context.Context, _ string) ([]wealth.Asset, error) {
	return s.assets, nil
}
func (s *stubAssetRepo) Create(_ context.Context, a wealth.Asset) (wealth.Asset, error) {
	return a, nil
}
func (s *stubAssetRepo) Update(_ context.Context, a wealth.Asset) (wealth.Asset, error) {
	return a, nil
}
func (s *stubAssetRepo) Delete(_ context.Context, _, _ string) error { return nil }

type stubLiabilityRepo struct{ liabilities []wealth.Liability }

func (s *stubLiabilityRepo) List(_ context.Context, _ string) ([]wealth.Liability, error) {
	return s.liabilities, nil
}
func (s *stubLiabilityRepo) Create(_ context.Context, l wealth.Liability) (wealth.Liability, error) {
	return l, nil
}
func (s *stubLiabilityRepo) Update(_ context.Context, l wealth.Liability) (wealth.Liability, error) {
	return l, nil
}
func (s *stubLiabilityRepo) Delete(_ context.Context, _, _ string) error { return nil }
func (s *stubLiabilityRepo) TotalBalance(_ context.Context, _ string) (float64, error) {
	var total float64
	for _, l := range s.liabilities {
		total += l.Balance
	}
	return total, nil
}

type stubTxRepo struct {
	cash    float64
	spent   float64
	budgets []finance.Budget
	txs     []finance.Transaction
}

func (s *stubTxRepo) List(_ context.Context, _, _, _, _ string, _, _ int) ([]finance.Transaction, error) {
	return s.txs, nil
}
func (s *stubTxRepo) Create(_ context.Context, t finance.Transaction) (finance.Transaction, error) {
	return t, nil
}
func (s *stubTxRepo) Delete(_ context.Context, _, _ string) error { return nil }
func (s *stubTxRepo) ListBudgets(_ context.Context, _ string) ([]finance.Budget, error) {
	return s.budgets, nil
}
func (s *stubTxRepo) UpsertBudget(_ context.Context, b finance.Budget) (finance.Budget, error) {
	return b, nil
}
func (s *stubTxRepo) SumByUser(_ context.Context, _ string) (float64, error) {
	return s.cash, nil
}
func (s *stubTxRepo) SumSpentThisMonth(_ context.Context, _ string) (float64, error) {
	return s.spent, nil
}

type stubGoalRepo struct {
	habitsTotal    int
	habitsDone     int
	goalsAvgProg   int
}

func (s *stubGoalRepo) List(_ context.Context, _ string) ([]goals.Goal, error) { return nil, nil }
func (s *stubGoalRepo) Create(_ context.Context, g goals.Goal) (goals.Goal, error) { return g, nil }
func (s *stubGoalRepo) Update(_ context.Context, g goals.Goal) (goals.Goal, error) { return g, nil }
func (s *stubGoalRepo) Delete(_ context.Context, _, _ string) error               { return nil }
func (s *stubGoalRepo) AddKeyResult(_ context.Context, kr goals.KeyResult) (goals.KeyResult, error) {
	return kr, nil
}
func (s *stubGoalRepo) UpdateKeyResult(_ context.Context, kr goals.KeyResult) (goals.KeyResult, error) {
	return kr, nil
}
func (s *stubGoalRepo) DeleteKeyResult(_ context.Context, _, _ string) error { return nil }
func (s *stubGoalRepo) HabitsSummary(_ context.Context, _ string) (int, int, error) {
	return s.habitsTotal, s.habitsDone, nil
}
func (s *stubGoalRepo) GoalsAvgProgress(_ context.Context, _ string) (int, error) {
	return s.goalsAvgProg, nil
}

type stubTrendsRepo struct{ snaps []trendsdomain.NetWorthSnapshot }

func (s *stubTrendsRepo) ListSnapshots(_ context.Context, _ string) ([]trendsdomain.NetWorthSnapshot, error) {
	return s.snaps, nil
}
func (s *stubTrendsRepo) UpsertSnapshot(_ context.Context, snap trendsdomain.NetWorthSnapshot) (trendsdomain.NetWorthSnapshot, error) {
	s.snaps = append(s.snaps, snap)
	return snap, nil
}
func (s *stubTrendsRepo) UpsertBenchmark(_ context.Context, _ trendsdomain.BenchmarkData) error {
	return nil
}
func (s *stubTrendsRepo) ListBenchmarks(_ context.Context, _ []string, _, _ string) ([]trendsdomain.BenchmarkData, error) {
	return nil, nil
}
func (s *stubTrendsRepo) LatestBankRates(_ context.Context) ([]trendsdomain.BankRate, error) {
	return nil, nil
}
func (s *stubTrendsRepo) UpsertBankRate(_ context.Context, _ trendsdomain.BankRate) error {
	return nil
}
func (s *stubTrendsRepo) ListNews(_ context.Context, _ int) ([]trendsdomain.NewsItem, error) {
	return nil, nil
}
func (s *stubTrendsRepo) UpsertNews(_ context.Context, _ []trendsdomain.NewsItem) error {
	return nil
}

// --- helpers ---

func newSvc(assets []wealth.Asset, liabilities []wealth.Liability, cash float64) (*Service, *stubTrendsRepo) {
	tr := &stubTrendsRepo{}
	svc := New(
		&stubAssetRepo{assets: assets},
		&stubLiabilityRepo{liabilities: liabilities},
		&stubTxRepo{cash: cash},
		&stubGoalRepo{},
		tr,
	)
	return svc, tr
}

// --- tests ---

func TestSummary_NetWorth(t *testing.T) {
	assets := []wealth.Asset{
		{CurrentValue: 200_000_000},
	}
	liabilities := []wealth.Liability{
		{Balance: 50_000_000},
	}
	svc, _ := newSvc(assets, liabilities, 10_000_000)

	sum, err := svc.Summary(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const want = 160_000_000.0
	if sum.NetWorth != want {
		t.Errorf("NetWorth = %v, want %v", sum.NetWorth, want)
	}
}

func TestSummary_HabitsAndGoals(t *testing.T) {
	tr := &stubTrendsRepo{}
	svc := New(
		&stubAssetRepo{},
		&stubLiabilityRepo{},
		&stubTxRepo{},
		&stubGoalRepo{habitsTotal: 5, habitsDone: 3, goalsAvgProg: 65},
		tr,
	)

	sum, err := svc.Summary(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sum.HabitsTotal != 5 {
		t.Errorf("HabitsTotal = %d, want 5", sum.HabitsTotal)
	}
	if sum.HabitsDoneToday != 3 {
		t.Errorf("HabitsDoneToday = %d, want 3", sum.HabitsDoneToday)
	}
	if sum.GoalsAvgProgress != 65 {
		t.Errorf("GoalsAvgProgress = %d, want 65", sum.GoalsAvgProgress)
	}
}

func TestSummary_NetWorthTrend_Sparkline(t *testing.T) {
	// Pre-seed 8 snapshots; service should return only last 6
	snaps := make([]trendsdomain.NetWorthSnapshot, 8)
	for i := range snaps {
		snaps[i] = trendsdomain.NetWorthSnapshot{NetWorth: float64((i + 1) * 10)}
	}
	tr := &stubTrendsRepo{snaps: snaps}
	svc := New(&stubAssetRepo{}, &stubLiabilityRepo{}, &stubTxRepo{}, &stubGoalRepo{}, tr)

	sum, err := svc.Summary(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// After upsert there are 9 snapshots; last 6 are indices 3-8 → values 40,50,60,70,80,90
	if len(sum.NetWorthTrend) != 6 {
		t.Errorf("NetWorthTrend length = %d, want 6", len(sum.NetWorthTrend))
	}
}

func TestSummary_RecentTx_NotNil(t *testing.T) {
	svc, _ := newSvc(nil, nil, 0)
	sum, err := svc.Summary(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum.RecentTx == nil {
		t.Error("RecentTx should not be nil")
	}
}

package dashboardsvc

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/finance"
	trendsdomain "github.com/chiutuanbinh/mylifeos/backend/internal/domain/trends"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	wealthsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/wealth"
)

// Summary is the dashboard aggregate output DTO.
type Summary struct {
	NetWorth         float64               `json:"net_worth"`
	NetWorthTrend    []float64             `json:"net_worth_trend"`
	HabitsTotal      int                   `json:"habits_total"`
	HabitsDoneToday  int                   `json:"habits_done_today"`
	GoalsAvgProgress int                   `json:"goals_avg_progress"`
	BudgetTotal      float64               `json:"budget_total"`
	BudgetSpent      float64               `json:"budget_spent"`
	RecentTx         []finance.Transaction `json:"recent_transactions"`
}

type Service struct {
	assets      repository.AssetRepo
	liabilities repository.LiabilityRepo
	txs         repository.TransactionRepo
	goals       repository.GoalRepo
	snapshots   repository.TrendsRepo
}

func New(
	assets repository.AssetRepo,
	liabilities repository.LiabilityRepo,
	txs repository.TransactionRepo,
	goals repository.GoalRepo,
	snapshots repository.TrendsRepo,
) *Service {
	return &Service{assets, liabilities, txs, goals, snapshots}
}

func (s *Service) Summary(ctx context.Context, userID string) (Summary, error) {
	var sum Summary

	// Habits + goals
	sum.HabitsTotal, sum.HabitsDoneToday, _ = s.goals.HabitsSummary(ctx, userID)
	sum.GoalsAvgProgress, _ = s.goals.GoalsAvgProgress(ctx, userID)

	// Budget
	budgets, _ := s.txs.ListBudgets(ctx, userID)
	for _, b := range budgets {
		sum.BudgetTotal += b.MonthlyLimit
	}
	sum.BudgetSpent, _ = s.txs.SumSpentThisMonth(ctx, userID)

	// Net worth
	assets, err := s.assets.List(ctx, userID)
	if err != nil {
		return sum, err
	}
	var assetsTotal float64
	for _, a := range assets {
		assetsTotal += a.CurrentValue
	}
	cash, _ := s.txs.SumByUser(ctx, userID)
	liabilities, _ := s.liabilities.List(ctx, userID)
	sum.NetWorth = wealthsvc.NetWorth(assetsTotal, cash, liabilities)

	// Upsert today's snapshot
	today := time.Now().Format("2006-01-02")
	s.snapshots.UpsertSnapshot(ctx, trendsdomain.NetWorthSnapshot{
		UserID:       userID,
		SnapshotDate: today,
		AssetsValue:  assetsTotal,
		CashPosition: cash,
		NetWorth:     sum.NetWorth,
	})

	// Sparkline — last 6 snapshots (ListSnapshots returns ASC order)
	snaps, err := s.snapshots.ListSnapshots(ctx, userID)
	if err != nil {
		return sum, err
	}
	start := len(snaps) - 6
	if start < 0 {
		start = 0
	}
	for _, sn := range snaps[start:] {
		sum.NetWorthTrend = append(sum.NetWorthTrend, sn.NetWorth)
	}
	if len(sum.NetWorthTrend) == 0 {
		sum.NetWorthTrend = []float64{sum.NetWorth}
	}

	// Recent transactions (most recent 6)
	sum.RecentTx, _ = s.txs.List(ctx, userID, "", "", "", 6, 0)
	if sum.RecentTx == nil {
		sum.RecentTx = []finance.Transaction{}
	}

	return sum, nil
}

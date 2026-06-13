package repo

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DashboardRepo interface {
	Summary(ctx context.Context, userID string) (models.DashboardSummary, error)
}

type pgDashboardRepo struct{ db *pgxpool.Pool }

func NewDashboardRepo(db *pgxpool.Pool) DashboardRepo {
	return &pgDashboardRepo{db}
}

func (r *pgDashboardRepo) Summary(ctx context.Context, userID string) (models.DashboardSummary, error) {
	var s models.DashboardSummary

	// Habits
	row := r.db.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN hl.done THEN 1 ELSE 0 END), 0)
		 FROM habits h
		 LEFT JOIN habit_logs hl ON hl.habit_id = h.id AND hl.logged_date = CURRENT_DATE
		 WHERE h.user_id = $1`, userID)
	row.Scan(&s.HabitsTotal, &s.HabitsDoneToday)

	// Goals avg progress computed from KRs (active only)
	row = r.db.QueryRow(ctx, `
		SELECT COALESCE(ROUND(AVG(
			CASE WHEN total = 0 THEN 0
			ELSE done_count::numeric / total * 100 END
		)), 0)
		FROM (
			SELECT g.id,
				COUNT(kr.id) AS total,
				COUNT(CASE WHEN kr.done THEN 1 END) AS done_count
			FROM goals g
			LEFT JOIN key_results kr ON kr.goal_id = g.id
			WHERE g.user_id = $1 AND g.status = 'active'
			GROUP BY g.id
		) sub`, userID)
	row.Scan(&s.GoalsAvgProgress)

	// Budget
	row = r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(monthly_limit), 0) FROM budgets WHERE user_id = $1`, userID)
	row.Scan(&s.BudgetTotal)

	row = r.db.QueryRow(ctx,
		`SELECT COALESCE(ABS(SUM(amount)), 0) FROM transactions
		 WHERE user_id = $1 AND amount < 0
		 AND date_trunc('month', date) = date_trunc('month', CURRENT_DATE)`, userID)
	row.Scan(&s.BudgetSpent)

	// Compute current assets value (with depreciation in Go)
	assetRows, err := r.db.Query(ctx,
		`SELECT value, purchase_value, depreciation_rate, purchased_at FROM assets WHERE user_id = $1`, userID)
	if err != nil {
		return s, err
	}
	var assetsTotal float64
	for assetRows.Next() {
		var value float64
		var purchaseValue *float64
		var depreciationRate float64
		var purchasedAt *time.Time
		assetRows.Scan(&value, &purchaseValue, &depreciationRate, &purchasedAt)
		if purchaseValue != nil && *purchaseValue > 0 {
			var pDate *string
			if purchasedAt != nil {
				s2 := purchasedAt.Format("2006-01-02")
				pDate = &s2
			}
			assetsTotal += computeCurrentValue(purchaseValue, depreciationRate, pDate)
		} else {
			assetsTotal += value
		}
	}
	assetRows.Close()

	// Cash position (all transactions sum)
	var cashPosition float64
	row = r.db.QueryRow(ctx, `SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE user_id = $1`, userID)
	row.Scan(&cashPosition)

	netWorth := assetsTotal + cashPosition
	s.NetWorth = netWorth

	// Upsert today's snapshot
	today := time.Now().Format("2006-01-02")
	r.db.Exec(ctx, `
		INSERT INTO net_worth_snapshots (user_id, snapshot_date, assets_value, cash_position, net_worth)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, snapshot_date)
		DO UPDATE SET assets_value=$3, cash_position=$4, net_worth=$5`,
		userID, today, assetsTotal, cashPosition, netWorth)

	// Sparkline: last 6 snapshots in chronological order
	snapRows, err := r.db.Query(ctx,
		`SELECT net_worth FROM net_worth_snapshots
		 WHERE user_id = $1 ORDER BY snapshot_date DESC LIMIT 6`, userID)
	if err != nil {
		return s, err
	}
	var trend []float64
	for snapRows.Next() {
		var nw float64
		snapRows.Scan(&nw)
		trend = append(trend, nw)
	}
	snapRows.Close()
	// reverse to chronological order
	for i, j := 0, len(trend)-1; i < j; i, j = i+1, j-1 {
		trend[i], trend[j] = trend[j], trend[i]
	}
	if len(trend) == 0 {
		trend = []float64{netWorth}
	}
	s.NetWorthTrend = trend

	// Recent transactions
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, date, description, category, amount, created_at
		 FROM transactions WHERE user_id = $1 ORDER BY date DESC, created_at DESC LIMIT 6`, userID)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var t models.Transaction
		var date time.Time
		rows.Scan(&t.ID, &t.UserID, &date, &t.Description, &t.Category, &t.Amount, &t.CreatedAt)
		t.Date = date.Format("2006-01-02")
		s.RecentTx = append(s.RecentTx, t)
	}
	if s.RecentTx == nil {
		s.RecentTx = []models.Transaction{}
	}

	return s, rows.Err()
}

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

	row := r.db.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN hl.done THEN 1 ELSE 0 END), 0)
		 FROM habits h
		 LEFT JOIN habit_logs hl ON hl.habit_id = h.id AND hl.logged_date = CURRENT_DATE
		 WHERE h.user_id = $1`, userID)
	row.Scan(&s.HabitsTotal, &s.HabitsDoneToday)

	row = r.db.QueryRow(ctx,
		`SELECT COALESCE(ROUND(AVG(progress)), 0) FROM goals WHERE user_id = $1`, userID)
	row.Scan(&s.GoalsAvgProgress)

	row = r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(monthly_limit), 0) FROM budgets WHERE user_id = $1`, userID)
	row.Scan(&s.BudgetTotal)

	row = r.db.QueryRow(ctx,
		`SELECT COALESCE(ABS(SUM(amount)), 0) FROM transactions
		 WHERE user_id = $1 AND amount < 0
		 AND date_trunc('month', date) = date_trunc('month', CURRENT_DATE)`, userID)
	row.Scan(&s.BudgetSpent)

	s.NetWorthTrend = []float64{110000, 115000, 118500, 121000, 125000, 0}
	row = r.db.QueryRow(ctx, `SELECT COALESCE(SUM(value), 0) FROM assets WHERE user_id = $1`, userID)
	var current float64
	row.Scan(&current)
	s.NetWorthTrend[5] = current

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

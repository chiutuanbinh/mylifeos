package postgres

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/finance"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgTransactionRepo struct{ db *pgxpool.Pool }

func NewTransactionRepo(db *pgxpool.Pool) repository.TransactionRepo { return &pgTransactionRepo{db} }

func (r *pgTransactionRepo) ListBudgets(ctx context.Context, userID string) ([]finance.Budget, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, category, monthly_limit, created_at FROM budgets WHERE user_id = $1 ORDER BY category`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []finance.Budget
	for rows.Next() {
		var b finance.Budget
		rows.Scan(&b.ID, &b.UserID, &b.Category, &b.MonthlyLimit, &b.CreatedAt)
		out = append(out, b)
	}
	if out == nil {
		out = []finance.Budget{}
	}
	return out, rows.Err()
}

func (r *pgTransactionRepo) UpsertBudget(ctx context.Context, b finance.Budget) (finance.Budget, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO budgets (user_id, category, monthly_limit)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, category) DO UPDATE SET monthly_limit = EXCLUDED.monthly_limit
		 RETURNING id, user_id, category, monthly_limit, created_at`,
		b.UserID, b.Category, b.MonthlyLimit)
	var out finance.Budget
	err := row.Scan(&out.ID, &out.UserID, &out.Category, &out.MonthlyLimit, &out.CreatedAt)
	return out, err
}

func (r *pgTransactionRepo) SumByUser(ctx context.Context, userID string) (float64, error) {
	var total float64
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE user_id = $1`, userID).Scan(&total)
	return total, err
}

func (r *pgTransactionRepo) SumSpentThisMonth(ctx context.Context, userID string) (float64, error) {
	var total float64
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(ABS(SUM(amount)), 0) FROM transactions
		 WHERE user_id = $1 AND amount < 0
		 AND date_trunc('month', date) = date_trunc('month', CURRENT_DATE)`, userID).Scan(&total)
	return total, err
}

func (r *pgTransactionRepo) DeleteBudget(ctx context.Context, userID, category string) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM budgets WHERE user_id = $1 AND category = $2`,
		userID, category,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrBudgetNotFound
	}
	return nil
}

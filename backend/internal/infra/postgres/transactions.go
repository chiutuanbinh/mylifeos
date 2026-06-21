package postgres

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/finance"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgTransactionRepo struct{ db *pgxpool.Pool }

func NewTransactionRepo(db *pgxpool.Pool) repository.TransactionRepo { return &pgTransactionRepo{db} }

func (r *pgTransactionRepo) List(ctx context.Context, userID, category, from, to string, limit, offset int) ([]finance.Transaction, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, date, description, category, amount, created_at
		 FROM transactions
		 WHERE user_id = $1
		   AND ($2 = '' OR category = $2)
		   AND ($3 = '' OR date >= $3::date)
		   AND ($4 = '' OR date <= $4::date)
		 ORDER BY date DESC, created_at DESC
		 LIMIT $5 OFFSET $6`,
		userID, category, from, to, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []finance.Transaction
	for rows.Next() {
		var t finance.Transaction
		var date time.Time
		rows.Scan(&t.ID, &t.UserID, &date, &t.Description, &t.Category, &t.Amount, &t.CreatedAt)
		t.Date = date.Format("2006-01-02")
		out = append(out, t)
	}
	if out == nil {
		out = []finance.Transaction{}
	}
	return out, rows.Err()
}

func (r *pgTransactionRepo) Create(ctx context.Context, t finance.Transaction) (finance.Transaction, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO transactions (user_id, date, description, category, amount)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, date, description, category, amount, created_at`,
		t.UserID, t.Date, t.Description, t.Category, t.Amount)
	var out finance.Transaction
	var date time.Time
	err := row.Scan(&out.ID, &out.UserID, &date, &out.Description, &out.Category, &out.Amount, &out.CreatedAt)
	out.Date = date.Format("2006-01-02")
	return out, err
}

func (r *pgTransactionRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM transactions WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

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

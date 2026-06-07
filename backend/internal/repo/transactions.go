package repo

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionRepo interface {
	List(ctx context.Context, userID, category, from, to string, limit, offset int) ([]models.Transaction, error)
	Create(ctx context.Context, t models.Transaction) (models.Transaction, error)
	Delete(ctx context.Context, id, userID string) error
	ListBudgets(ctx context.Context, userID string) ([]models.Budget, error)
	UpsertBudget(ctx context.Context, b models.Budget) (models.Budget, error)
}

type pgTransactionRepo struct{ db *pgxpool.Pool }

func NewTransactionRepo(db *pgxpool.Pool) TransactionRepo { return &pgTransactionRepo{db} }

func (r *pgTransactionRepo) List(ctx context.Context, userID, category, from, to string, limit, offset int) ([]models.Transaction, error) {
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
	var out []models.Transaction
	for rows.Next() {
		var t models.Transaction
		rows.Scan(&t.ID, &t.UserID, &t.Date, &t.Description, &t.Category, &t.Amount, &t.CreatedAt)
		out = append(out, t)
	}
	if out == nil {
		out = []models.Transaction{}
	}
	return out, rows.Err()
}

func (r *pgTransactionRepo) Create(ctx context.Context, t models.Transaction) (models.Transaction, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO transactions (user_id, date, description, category, amount)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, date, description, category, amount, created_at`,
		t.UserID, t.Date, t.Description, t.Category, t.Amount)
	var out models.Transaction
	err := row.Scan(&out.ID, &out.UserID, &out.Date, &out.Description, &out.Category, &out.Amount, &out.CreatedAt)
	return out, err
}

func (r *pgTransactionRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM transactions WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (r *pgTransactionRepo) ListBudgets(ctx context.Context, userID string) ([]models.Budget, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, category, monthly_limit, created_at FROM budgets WHERE user_id = $1 ORDER BY category`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Budget
	for rows.Next() {
		var b models.Budget
		rows.Scan(&b.ID, &b.UserID, &b.Category, &b.MonthlyLimit, &b.CreatedAt)
		out = append(out, b)
	}
	if out == nil {
		out = []models.Budget{}
	}
	return out, rows.Err()
}

func (r *pgTransactionRepo) UpsertBudget(ctx context.Context, b models.Budget) (models.Budget, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO budgets (user_id, category, monthly_limit)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, category) DO UPDATE SET monthly_limit = EXCLUDED.monthly_limit
		 RETURNING id, user_id, category, monthly_limit, created_at`,
		b.UserID, b.Category, b.MonthlyLimit)
	var out models.Budget
	err := row.Scan(&out.ID, &out.UserID, &out.Category, &out.MonthlyLimit, &out.CreatedAt)
	return out, err
}

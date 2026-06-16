package postgres

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/wealth"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgLiabilityRepo struct{ db *pgxpool.Pool }

func NewLiabilityRepo(db *pgxpool.Pool) repository.LiabilityRepo { return &pgLiabilityRepo{db} }

func scanLiability(row interface{ Scan(...any) error }) (wealth.Liability, error) {
	var l wealth.Liability
	var startedAt, dueAt *time.Time
	err := row.Scan(&l.ID, &l.UserID, &l.Name, &l.Category, &l.Balance,
		&l.OriginalPrincipal, &l.InterestRate, &startedAt, &dueAt, &l.Notes)
	if startedAt != nil {
		s := startedAt.Format("2006-01-02")
		l.StartedAt = &s
	}
	if dueAt != nil {
		s := dueAt.Format("2006-01-02")
		l.DueAt = &s
	}
	return l, err
}

const liabilityCols = `id, user_id, name, category, balance, original_principal, interest_rate, started_at, due_at, notes`

func (r *pgLiabilityRepo) List(ctx context.Context, userID string) ([]wealth.Liability, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+liabilityCols+` FROM liabilities WHERE user_id=$1 ORDER BY category, name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []wealth.Liability
	for rows.Next() {
		l, err := scanLiability(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	if out == nil {
		out = []wealth.Liability{}
	}
	return out, rows.Err()
}

func (r *pgLiabilityRepo) Create(ctx context.Context, l wealth.Liability) (wealth.Liability, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO liabilities (user_id, name, category, balance, original_principal, interest_rate, started_at, due_at, notes)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING `+liabilityCols,
		l.UserID, l.Name, l.Category, l.Balance, l.OriginalPrincipal, l.InterestRate, l.StartedAt, l.DueAt, l.Notes)
	return scanLiability(row)
}

func (r *pgLiabilityRepo) Update(ctx context.Context, l wealth.Liability) (wealth.Liability, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE liabilities SET name=$1, category=$2, balance=$3, original_principal=$4,
		 interest_rate=$5, started_at=$6, due_at=$7, notes=$8
		 WHERE id=$9 AND user_id=$10
		 RETURNING `+liabilityCols,
		l.Name, l.Category, l.Balance, l.OriginalPrincipal, l.InterestRate, l.StartedAt, l.DueAt, l.Notes, l.ID, l.UserID)
	return scanLiability(row)
}

func (r *pgLiabilityRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM liabilities WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

func (r *pgLiabilityRepo) TotalBalance(ctx context.Context, userID string) (float64, error) {
	var total float64
	err := r.db.QueryRow(ctx, `SELECT COALESCE(SUM(balance),0) FROM liabilities WHERE user_id=$1`, userID).Scan(&total)
	return total, err
}

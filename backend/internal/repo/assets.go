package repo

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AssetRepo interface {
	List(ctx context.Context, userID string) ([]models.Asset, error)
	Create(ctx context.Context, a models.Asset) (models.Asset, error)
	Update(ctx context.Context, a models.Asset) (models.Asset, error)
	Delete(ctx context.Context, id, userID string) error
}

type pgAssetRepo struct{ db *pgxpool.Pool }

func NewAssetRepo(db *pgxpool.Pool) AssetRepo { return &pgAssetRepo{db} }

func (r *pgAssetRepo) List(ctx context.Context, userID string) ([]models.Asset, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, name, category, value, purchased_at, notes
		 FROM assets WHERE user_id = $1 ORDER BY category, name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Asset
	for rows.Next() {
		var a models.Asset
		var purchasedAt *time.Time
		rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Category, &a.Value, &purchasedAt, &a.Notes)
		a.PurchasedAt = nullDateString(purchasedAt)
		out = append(out, a)
	}
	if out == nil {
		out = []models.Asset{}
	}
	return out, rows.Err()
}

func (r *pgAssetRepo) Create(ctx context.Context, a models.Asset) (models.Asset, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO assets (user_id, name, category, value, purchased_at, notes)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, name, category, value, purchased_at, notes`,
		a.UserID, a.Name, a.Category, a.Value, a.PurchasedAt, a.Notes)
	var out models.Asset
	var purchasedAt *time.Time
	err := row.Scan(&out.ID, &out.UserID, &out.Name, &out.Category, &out.Value, &purchasedAt, &out.Notes)
	out.PurchasedAt = nullDateString(purchasedAt)
	return out, err
}

func (r *pgAssetRepo) Update(ctx context.Context, a models.Asset) (models.Asset, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE assets SET name=$1, category=$2, value=$3, purchased_at=$4, notes=$5
		 WHERE id=$6 AND user_id=$7
		 RETURNING id, user_id, name, category, value, purchased_at, notes`,
		a.Name, a.Category, a.Value, a.PurchasedAt, a.Notes, a.ID, a.UserID)
	var out models.Asset
	var purchasedAt *time.Time
	err := row.Scan(&out.ID, &out.UserID, &out.Name, &out.Category, &out.Value, &purchasedAt, &out.Notes)
	out.PurchasedAt = nullDateString(purchasedAt)
	return out, err
}

func (r *pgAssetRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM assets WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

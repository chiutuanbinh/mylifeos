package postgres

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/wealth"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	wealthsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/wealth"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgAssetRepo struct{ db *pgxpool.Pool }

func NewAssetRepo(db *pgxpool.Pool) repository.AssetRepo { return &pgAssetRepo{db} }

func scanAsset(row interface {
	Scan(...any) error
}) (wealth.Asset, error) {
	var a wealth.Asset
	var purchasedAt *time.Time
	err := row.Scan(&a.ID, &a.UserID, &a.Name, &a.Category, &a.Value, &purchasedAt, &a.Notes, &a.PurchaseValue, &a.DepreciationRate)
	if purchasedAt != nil {
		s := purchasedAt.Format("2006-01-02")
		a.PurchasedAt = &s
	}
	if a.PurchaseValue != nil {
		a.CurrentValue = wealthsvc.CurrentValue(a)
	} else {
		a.CurrentValue = a.Value
	}
	return a, err
}

func (r *pgAssetRepo) List(ctx context.Context, userID string) ([]wealth.Asset, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, name, category, value, purchased_at, notes, purchase_value, depreciation_rate
		 FROM assets WHERE user_id = $1 ORDER BY category, name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []wealth.Asset
	for rows.Next() {
		a, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if out == nil {
		out = []wealth.Asset{}
	}
	return out, rows.Err()
}

func (r *pgAssetRepo) Create(ctx context.Context, a wealth.Asset) (wealth.Asset, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO assets (user_id, name, category, value, purchased_at, notes, purchase_value, depreciation_rate)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, user_id, name, category, value, purchased_at, notes, purchase_value, depreciation_rate`,
		a.UserID, a.Name, a.Category, a.Value, a.PurchasedAt, a.Notes, a.PurchaseValue, a.DepreciationRate)
	return scanAsset(row)
}

func (r *pgAssetRepo) Update(ctx context.Context, a wealth.Asset) (wealth.Asset, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE assets SET name=$1, category=$2, value=$3, purchased_at=$4, notes=$5, purchase_value=$6, depreciation_rate=$7
		 WHERE id=$8 AND user_id=$9
		 RETURNING id, user_id, name, category, value, purchased_at, notes, purchase_value, depreciation_rate`,
		a.Name, a.Category, a.Value, a.PurchasedAt, a.Notes, a.PurchaseValue, a.DepreciationRate, a.ID, a.UserID)
	return scanAsset(row)
}

func (r *pgAssetRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM assets WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

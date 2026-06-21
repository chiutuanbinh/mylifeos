package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
)

type pgAccountRepo struct{ db *pgxpool.Pool }

func NewAccountRepo(db *pgxpool.Pool) repository.AccountRepo {
	return &pgAccountRepo{db: db}
}

func (r *pgAccountRepo) Save(ctx context.Context, a *accounting.Account) error {
	var parentID *string
	if a.ParentID() != nil {
		s := string(*a.ParentID())
		parentID = &s
	}
	var (
		purchaseValue    *decimal.Decimal
		purchasedAt      *time.Time
		depreciationRate *decimal.Decimal
		assetNotes       *string
	)
	if m := a.AssetMeta(); m != nil {
		purchaseValue = m.PurchaseValue
		purchasedAt = m.PurchasedAt
		depreciationRate = m.DepreciationRate
		if m.Notes != "" {
			assetNotes = &m.Notes
		}
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO accounts (id, user_id, parent_id, name, type, currency, is_group, archived, sort_order,
		                      purchase_value, purchased_at, depreciation_rate, asset_notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (id) DO UPDATE SET
			parent_id=$3, name=$4, type=$5, currency=$6, is_group=$7, archived=$8, sort_order=$9,
			purchase_value=$10, purchased_at=$11, depreciation_rate=$12, asset_notes=$13`,
		string(a.ID()), a.UserID(), parentID, a.Name(),
		string(a.Type()), a.Currency(), a.IsGroup(), a.Archived(), a.SortOrder(),
		purchaseValue, purchasedAt, depreciationRate, assetNotes,
	)
	return err
}

func (r *pgAccountRepo) FindByUser(ctx context.Context, userID string) ([]*accounting.Account, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, parent_id, name, type, currency, is_group, archived, sort_order,
		       purchase_value, purchased_at, depreciation_rate, asset_notes
		FROM accounts WHERE user_id = $1 AND archived = false ORDER BY sort_order, name`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAccounts(rows)
}

func (r *pgAccountRepo) FindByID(ctx context.Context, id accounting.AccountID) (*accounting.Account, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, parent_id, name, type, currency, is_group, archived, sort_order,
		       purchase_value, purchased_at, depreciation_rate, asset_notes
		FROM accounts WHERE id = $1`,
		string(id),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accounts, err := scanAccounts(rows)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, repository.ErrAccountNotFound
	}
	return accounts[0], nil
}

func (r *pgAccountRepo) FindByNameAndType(ctx context.Context, userID, name string, t accounting.AccountType) (*accounting.Account, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, parent_id, name, type, currency, is_group, archived, sort_order,
		       purchase_value, purchased_at, depreciation_rate, asset_notes
		FROM accounts WHERE user_id = $1 AND name = $2 AND type = $3 AND archived = false LIMIT 1`,
		userID, name, string(t),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accounts, err := scanAccounts(rows)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, repository.ErrAccountNotFound
	}
	return accounts[0], nil
}

func scanAccounts(rows pgx.Rows) ([]*accounting.Account, error) {
	var result []*accounting.Account
	for rows.Next() {
		var (
			id, userID, name, acctType, currency string
			parentID                             *string
			isGroup, archived                    bool
			sortOrder                            int
			purchaseValue                        *decimal.Decimal
			purchasedAt                          *time.Time
			depreciationRate                     *decimal.Decimal
			assetNotes                           *string
		)
		if err := rows.Scan(
			&id, &userID, &parentID, &name, &acctType, &currency, &isGroup, &archived, &sortOrder,
			&purchaseValue, &purchasedAt, &depreciationRate, &assetNotes,
		); err != nil {
			return nil, err
		}
		a := accounting.ReconstituteAccount(
			id, userID, parentID, name,
			accounting.AccountType(acctType), currency, isGroup, archived, sortOrder,
		)
		if purchaseValue != nil || purchasedAt != nil || depreciationRate != nil || assetNotes != nil {
			meta := &accounting.AssetMeta{
				PurchaseValue:    purchaseValue,
				PurchasedAt:      purchasedAt,
				DepreciationRate: depreciationRate,
			}
			if assetNotes != nil {
				meta.Notes = *assetNotes
			}
			a.AttachAssetMeta(meta)
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

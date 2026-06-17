package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

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
	_, err := r.db.Exec(ctx, `
		INSERT INTO accounts (id, user_id, parent_id, name, type, currency, is_group, archived, sort_order)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (id) DO UPDATE SET
			parent_id=$3, name=$4, type=$5, currency=$6, is_group=$7, archived=$8, sort_order=$9`,
		string(a.ID()), a.UserID(), parentID, a.Name(),
		string(a.Type()), a.Currency(), a.IsGroup(), a.Archived(), a.SortOrder(),
	)
	return err
}

func (r *pgAccountRepo) FindByUser(ctx context.Context, userID string) ([]*accounting.Account, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, parent_id, name, type, currency, is_group, archived, sort_order
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
		SELECT id, user_id, parent_id, name, type, currency, is_group, archived, sort_order
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
		return nil, errors.New("account not found")
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
		)
		if err := rows.Scan(&id, &userID, &parentID, &name, &acctType, &currency, &isGroup, &archived, &sortOrder); err != nil {
			return nil, err
		}
		result = append(result, accounting.ReconstitueAccount(
			id, userID, parentID, name,
			accounting.AccountType(acctType), currency, isGroup, archived, sortOrder,
		))
	}
	return result, rows.Err()
}

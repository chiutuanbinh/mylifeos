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

type pgJournalRepo struct{ db *pgxpool.Pool }

func NewJournalRepo(db *pgxpool.Pool) repository.JournalRepo {
	return &pgJournalRepo{db: db}
}

func (r *pgJournalRepo) Save(ctx context.Context, e *accounting.JournalEntry) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO journal_entries (id, user_id, date, description, memo)
		VALUES ($1,$2,$3,$4,$5)`,
		string(e.ID()), e.UserID(), e.Date(), e.Description(), e.Memo(),
	)
	if err != nil {
		return err
	}

	for _, l := range e.Lines() {
		_, err = tx.Exec(ctx, `
			INSERT INTO journal_lines (id, entry_id, account_id, amount, currency, side)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			l.ID(), string(e.ID()), string(l.AccountID()),
			l.Money().Amount, l.Money().Currency, string(l.Side()),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *pgJournalRepo) FindByUser(ctx context.Context, userID string, from, to time.Time) ([]*accounting.JournalEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT e.id, e.user_id, e.date, e.description, e.memo,
		       l.id, l.account_id, l.amount, l.currency, l.side
		FROM journal_entries e
		JOIN journal_lines l ON l.entry_id = e.id
		WHERE e.user_id = $1 AND ($2::date IS NULL OR e.date >= $2) AND ($3::date IS NULL OR e.date <= $3)
		ORDER BY e.date DESC, e.id, l.id`,
		userID, nullDate(from), nullDate(to),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return reconstituteEntries(rows)
}

func nullDate(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func reconstituteEntries(rows pgx.Rows) ([]*accounting.JournalEntry, error) {
	entries := map[string]*accounting.JournalEntry{}
	order := []string{}

	for rows.Next() {
		var (
			eID, eUserID, eDesc, eMemo string
			eDate                      time.Time
			lID, lAcctID, lCurrency    string
			lAmount                    decimal.Decimal
			lSide                      string
		)
		if err := rows.Scan(&eID, &eUserID, &eDate, &eDesc, &eMemo,
			&lID, &lAcctID, &lAmount, &lCurrency, &lSide); err != nil {
			return nil, err
		}
		if _, exists := entries[eID]; !exists {
			entries[eID] = accounting.ReconstitueEntry(eID, eUserID, eDate, eDesc, eMemo)
			order = append(order, eID)
		}
		entries[eID].ReconstituteLine(
			lID,
			accounting.AccountID(lAcctID),
			accounting.Money{Amount: lAmount, Currency: lCurrency},
			accounting.Side(lSide),
		)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]*accounting.JournalEntry, len(order))
	for i, id := range order {
		result[i] = entries[id]
	}
	return result, nil
}

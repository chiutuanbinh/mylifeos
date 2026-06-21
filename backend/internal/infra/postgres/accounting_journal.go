package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	entries, err := reconstituteEntries(rows)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return entries, nil
	}

	// load goal links for all entries in one query
	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = string(e.ID())
	}
	goalRows, err := r.db.Query(ctx,
		`SELECT entry_id, goal_id FROM journal_entry_goals WHERE entry_id = ANY($1)`,
		ids,
	)
	if err != nil {
		return nil, err
	}
	defer goalRows.Close()
	goalMap := map[string][]string{}
	for goalRows.Next() {
		var entryID, goalID string
		if err := goalRows.Scan(&entryID, &goalID); err != nil {
			return nil, err
		}
		goalMap[entryID] = append(goalMap[entryID], goalID)
	}
	if err := goalRows.Err(); err != nil {
		return nil, err
	}
	for _, e := range entries {
		if gids, ok := goalMap[string(e.ID())]; ok {
			e.SetGoalIDs(gids)
		}
	}
	return entries, nil
}

func (r *pgJournalRepo) SaveGoalLinks(ctx context.Context, entryID, userID string, goalIDs []string) error {
	if len(goalIDs) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, gid := range goalIDs {
		batch.Queue(
			`INSERT INTO journal_entry_goals (entry_id, goal_id, user_id) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
			entryID, gid, userID,
		)
	}
	br := r.db.SendBatch(ctx, batch)
	defer br.Close()
	for range goalIDs {
		if _, err := br.Exec(); err != nil {
			// ignore FK violations (goal was deleted) — best-effort
			if !isFKViolation(err) {
				return err
			}
		}
	}
	return nil
}

// isFKViolation returns true for PostgreSQL error code 23503 (foreign_key_violation).
func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
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
			entries[eID] = accounting.ReconstituteEntry(eID, eUserID, eDate, eDesc, eMemo)
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

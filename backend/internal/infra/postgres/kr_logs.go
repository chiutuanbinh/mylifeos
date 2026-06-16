package postgres

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/goals"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgKRLogRepo struct{ db *pgxpool.Pool }

func NewKRLogRepo(db *pgxpool.Pool) repository.KRLogRepo { return &pgKRLogRepo{db} }

func scanKRLog(row interface{ Scan(...any) error }) (goals.KRLog, error) {
	var l goals.KRLog
	var loggedDate time.Time
	err := row.Scan(&l.ID, &l.KRID, &l.UserID, &loggedDate, &l.Done)
	l.LoggedDate = loggedDate.Format("2006-01-02")
	return l, err
}

func (r *pgKRLogRepo) GetLogs(ctx context.Context, userID, date string) ([]goals.KRLog, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	rows, err := r.db.Query(ctx,
		`SELECT kl.id, kl.kr_id, kl.user_id, kl.logged_date, kl.done
		 FROM kr_logs kl
		 JOIN key_results kr ON kr.id = kl.kr_id
		 WHERE kl.user_id = $1 AND kl.logged_date = $2::date AND kr.recurring = TRUE`,
		userID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []goals.KRLog
	for rows.Next() {
		l, err := scanKRLog(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	if out == nil {
		out = []goals.KRLog{}
	}
	return out, rows.Err()
}

func (r *pgKRLogRepo) GetLogRange(ctx context.Context, krID, userID, from, to string) ([]goals.KRLog, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, kr_id, user_id, logged_date, done
		 FROM kr_logs
		 WHERE kr_id = $1 AND user_id = $2 AND logged_date BETWEEN $3::date AND $4::date
		 ORDER BY logged_date`,
		krID, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []goals.KRLog
	for rows.Next() {
		l, err := scanKRLog(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	if out == nil {
		out = []goals.KRLog{}
	}
	return out, rows.Err()
}

func (r *pgKRLogRepo) ToggleLog(ctx context.Context, krID, userID, date string) (goals.KRLog, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	row := r.db.QueryRow(ctx,
		`INSERT INTO kr_logs (kr_id, user_id, logged_date, done)
		 VALUES ($1, $2, $3::date, TRUE)
		 ON CONFLICT (kr_id, logged_date)
		 DO UPDATE SET done = NOT kr_logs.done
		 RETURNING id, kr_id, user_id, logged_date, done`,
		krID, userID, date)
	return scanKRLog(row)
}

package postgres

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/calendar"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgEventRepo struct{ db *pgxpool.Pool }

func NewEventRepo(db *pgxpool.Pool) repository.EventRepo { return &pgEventRepo{db} }

func scanEvent(row interface{ Scan(...any) error }) (calendar.Event, error) {
	var e calendar.Event
	var startAt, endAt time.Time
	err := row.Scan(&e.ID, &e.UserID, &e.Title, &startAt, &endAt, &e.Color, &e.AllDay, &e.GoogleEventID)
	if err != nil {
		return e, err
	}
	e.StartAt = startAt.UTC().Format(time.RFC3339)
	e.EndAt = endAt.UTC().Format(time.RFC3339)
	return e, nil
}

func (r *pgEventRepo) List(ctx context.Context, userID, from, to string) ([]calendar.Event, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, title, start_at, end_at, color, all_day, google_event_id
		 FROM events
		 WHERE user_id = $1
		   AND ($2 = '' OR start_at >= $2::timestamptz)
		   AND ($3 = '' OR end_at   <= $3::timestamptz)
		 ORDER BY start_at`, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []calendar.Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	if out == nil {
		out = []calendar.Event{}
	}
	return out, rows.Err()
}

func (r *pgEventRepo) Create(ctx context.Context, e calendar.Event) (calendar.Event, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO events (user_id, title, start_at, end_at, color, all_day)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, title, start_at, end_at, color, all_day, google_event_id`,
		e.UserID, e.Title, e.StartAt, e.EndAt, e.Color, e.AllDay)
	return scanEvent(row)
}

func (r *pgEventRepo) Update(ctx context.Context, e calendar.Event) (calendar.Event, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE events SET title=$1, start_at=$2, end_at=$3, color=$4, all_day=$5
		 WHERE id=$6 AND user_id=$7
		 RETURNING id, user_id, title, start_at, end_at, color, all_day, google_event_id`,
		e.Title, e.StartAt, e.EndAt, e.Color, e.AllDay, e.ID, e.UserID)
	return scanEvent(row)
}

func (r *pgEventRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM events WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

func (r *pgEventRepo) UpsertFromGoogle(ctx context.Context, userID string, events []calendar.Event) (int, error) {
	count := 0
	for _, e := range events {
		if e.GoogleEventID == nil {
			continue
		}
		_, err := r.db.Exec(ctx,
			`INSERT INTO events (user_id, title, start_at, end_at, color, all_day, google_event_id)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 ON CONFLICT (google_event_id) WHERE google_event_id IS NOT NULL
			 DO UPDATE SET title=$2, start_at=$3, end_at=$4, all_day=$6`,
			userID, e.Title, e.StartAt, e.EndAt, e.Color, e.AllDay, *e.GoogleEventID)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

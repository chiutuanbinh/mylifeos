package repo

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventRepo interface {
	List(ctx context.Context, userID, from, to string) ([]models.Event, error)
	Create(ctx context.Context, e models.Event) (models.Event, error)
	Update(ctx context.Context, e models.Event) (models.Event, error)
	Delete(ctx context.Context, id, userID string) error
}

type pgEventRepo struct{ db *pgxpool.Pool }

func NewEventRepo(db *pgxpool.Pool) EventRepo { return &pgEventRepo{db} }

func (r *pgEventRepo) List(ctx context.Context, userID, from, to string) ([]models.Event, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, title, start_at, end_at, color, all_day
		 FROM events
		 WHERE user_id = $1
		   AND ($2 = '' OR start_at >= $2::timestamptz)
		   AND ($3 = '' OR end_at   <= $3::timestamptz)
		 ORDER BY start_at`, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Event
	for rows.Next() {
		var e models.Event
		rows.Scan(&e.ID, &e.UserID, &e.Title, &e.StartAt, &e.EndAt, &e.Color, &e.AllDay)
		out = append(out, e)
	}
	if out == nil {
		out = []models.Event{}
	}
	return out, rows.Err()
}

func (r *pgEventRepo) Create(ctx context.Context, e models.Event) (models.Event, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO events (user_id, title, start_at, end_at, color, all_day)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, title, start_at, end_at, color, all_day`,
		e.UserID, e.Title, e.StartAt, e.EndAt, e.Color, e.AllDay)
	var out models.Event
	err := row.Scan(&out.ID, &out.UserID, &out.Title, &out.StartAt, &out.EndAt, &out.Color, &out.AllDay)
	return out, err
}

func (r *pgEventRepo) Update(ctx context.Context, e models.Event) (models.Event, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE events SET title=$1, start_at=$2, end_at=$3, color=$4, all_day=$5
		 WHERE id=$6 AND user_id=$7
		 RETURNING id, user_id, title, start_at, end_at, color, all_day`,
		e.Title, e.StartAt, e.EndAt, e.Color, e.AllDay, e.ID, e.UserID)
	var out models.Event
	err := row.Scan(&out.ID, &out.UserID, &out.Title, &out.StartAt, &out.EndAt, &out.Color, &out.AllDay)
	return out, err
}

func (r *pgEventRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM events WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

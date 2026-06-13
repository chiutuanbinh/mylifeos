package repo

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HabitRepo interface {
	List(ctx context.Context, userID string) ([]models.Habit, error)
	Create(ctx context.Context, h models.Habit) (models.Habit, error)
	Delete(ctx context.Context, id, userID string) error
	GetLogs(ctx context.Context, userID, date string) ([]models.HabitLog, error)
	ToggleLog(ctx context.Context, habitID, userID, date string) (models.HabitLog, error)
}

type pgHabitRepo struct{ db *pgxpool.Pool }

func NewHabitRepo(db *pgxpool.Pool) HabitRepo { return &pgHabitRepo{db} }

func (r *pgHabitRepo) List(ctx context.Context, userID string) ([]models.Habit, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, name, icon, created_at FROM habits WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Habit
	for rows.Next() {
		var h models.Habit
		rows.Scan(&h.ID, &h.UserID, &h.Name, &h.Icon, &h.CreatedAt)
		out = append(out, h)
	}
	if out == nil {
		out = []models.Habit{}
	}
	return out, rows.Err()
}

func (r *pgHabitRepo) Create(ctx context.Context, h models.Habit) (models.Habit, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO habits (user_id, name, icon) VALUES ($1, $2, $3)
		 RETURNING id, user_id, name, icon, created_at`,
		h.UserID, h.Name, h.Icon)
	var out models.Habit
	err := row.Scan(&out.ID, &out.UserID, &out.Name, &out.Icon, &out.CreatedAt)
	return out, err
}

func (r *pgHabitRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM habits WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (r *pgHabitRepo) GetLogs(ctx context.Context, userID, date string) ([]models.HabitLog, error) {
	if date == "" {
		date = "CURRENT_DATE"
	}
	rows, err := r.db.Query(ctx,
		`SELECT hl.id, hl.habit_id, hl.user_id, hl.logged_date, hl.done
		 FROM habit_logs hl
		 JOIN habits h ON h.id = hl.habit_id
		 WHERE hl.user_id = $1 AND hl.logged_date = $2::date`, userID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.HabitLog
	for rows.Next() {
		var l models.HabitLog
		var loggedDate time.Time
		rows.Scan(&l.ID, &l.HabitID, &l.UserID, &loggedDate, &l.Done)
		l.LoggedDate = loggedDate.Format("2006-01-02")
		out = append(out, l)
	}
	if out == nil {
		out = []models.HabitLog{}
	}
	return out, rows.Err()
}

func (r *pgHabitRepo) ToggleLog(ctx context.Context, habitID, userID, date string) (models.HabitLog, error) {
	if date == "" {
		date = "CURRENT_DATE"
	}
	row := r.db.QueryRow(ctx,
		`INSERT INTO habit_logs (habit_id, user_id, logged_date, done)
		 VALUES ($1, $2, $3::date, true)
		 ON CONFLICT (habit_id, logged_date)
		 DO UPDATE SET done = NOT habit_logs.done
		 RETURNING id, habit_id, user_id, logged_date, done`,
		habitID, userID, date)
	var out models.HabitLog
	var loggedDate time.Time
	err := row.Scan(&out.ID, &out.HabitID, &out.UserID, &loggedDate, &out.Done)
	out.LoggedDate = loggedDate.Format("2006-01-02")
	return out, err
}

package repo

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GoalRepo interface {
	List(ctx context.Context, userID string) ([]models.Goal, error)
	Create(ctx context.Context, g models.Goal) (models.Goal, error)
	Update(ctx context.Context, g models.Goal) (models.Goal, error)
	Delete(ctx context.Context, id, userID string) error
	AddKeyResult(ctx context.Context, kr models.KeyResult) (models.KeyResult, error)
	UpdateKeyResult(ctx context.Context, kr models.KeyResult) (models.KeyResult, error)
	DeleteKeyResult(ctx context.Context, krID, userID string) error
}

func nullDateString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

func computeProgress(krs []models.KeyResult) int {
	if len(krs) == 0 {
		return 0
	}
	done := 0
	for _, kr := range krs {
		if kr.Done {
			done++
		}
	}
	return int(float64(done) / float64(len(krs)) * 100)
}

type pgGoalRepo struct{ db *pgxpool.Pool }

func NewGoalRepo(db *pgxpool.Pool) GoalRepo { return &pgGoalRepo{db} }

func (r *pgGoalRepo) List(ctx context.Context, userID string) ([]models.Goal, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, name, description, target_date, progress, color, status, created_at
		 FROM goals WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		var targetDate *time.Time
		rows.Scan(&g.ID, &g.UserID, &g.Name, &g.Description, &targetDate, &g.Progress, &g.Color, &g.Status, &g.CreatedAt)
		g.TargetDate = nullDateString(targetDate)
		goals = append(goals, g)
	}
	if goals == nil {
		goals = []models.Goal{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i, g := range goals {
		krows, err := r.db.Query(ctx,
			`SELECT id, goal_id, user_id, description, done FROM key_results WHERE goal_id = $1`, g.ID)
		if err != nil {
			return nil, err
		}
		var krs []models.KeyResult
		for krows.Next() {
			var kr models.KeyResult
			krows.Scan(&kr.ID, &kr.GoalID, &kr.UserID, &kr.Description, &kr.Done)
			krs = append(krs, kr)
		}
		krows.Close()
		if krs == nil {
			krs = []models.KeyResult{}
		}
		goals[i].KeyResults = krs
		goals[i].Progress = computeProgress(krs)
	}
	return goals, nil
}

func (r *pgGoalRepo) Create(ctx context.Context, g models.Goal) (models.Goal, error) {
	if g.Status == "" {
		g.Status = "active"
	}
	row := r.db.QueryRow(ctx,
		`INSERT INTO goals (user_id, name, description, target_date, progress, color, status)
		 VALUES ($1, $2, $3, $4, 0, $5, $6)
		 RETURNING id, user_id, name, description, target_date, progress, color, status, created_at`,
		g.UserID, g.Name, g.Description, g.TargetDate, g.Color, g.Status)
	var out models.Goal
	var targetDate *time.Time
	err := row.Scan(&out.ID, &out.UserID, &out.Name, &out.Description, &targetDate, &out.Progress, &out.Color, &out.Status, &out.CreatedAt)
	out.TargetDate = nullDateString(targetDate)
	out.KeyResults = []models.KeyResult{}
	return out, err
}

func (r *pgGoalRepo) Update(ctx context.Context, g models.Goal) (models.Goal, error) {
	if g.Status == "" {
		g.Status = "active"
	}
	row := r.db.QueryRow(ctx,
		`UPDATE goals SET name=$1, description=$2, target_date=$3, color=$4, status=$5
		 WHERE id=$6 AND user_id=$7
		 RETURNING id, user_id, name, description, target_date, progress, color, status, created_at`,
		g.Name, g.Description, g.TargetDate, g.Color, g.Status, g.ID, g.UserID)
	var out models.Goal
	var targetDate *time.Time
	err := row.Scan(&out.ID, &out.UserID, &out.Name, &out.Description, &targetDate, &out.Progress, &out.Color, &out.Status, &out.CreatedAt)
	out.TargetDate = nullDateString(targetDate)
	return out, err
}

func (r *pgGoalRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM goals WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

func (r *pgGoalRepo) AddKeyResult(ctx context.Context, kr models.KeyResult) (models.KeyResult, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO key_results (goal_id, user_id, description, done)
		 VALUES ($1, $2, $3, false)
		 RETURNING id, goal_id, user_id, description, done`,
		kr.GoalID, kr.UserID, kr.Description)
	var out models.KeyResult
	err := row.Scan(&out.ID, &out.GoalID, &out.UserID, &out.Description, &out.Done)
	return out, err
}

func (r *pgGoalRepo) UpdateKeyResult(ctx context.Context, kr models.KeyResult) (models.KeyResult, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE key_results SET description=$1, done=$2
		 WHERE id=$3 AND user_id=$4
		 RETURNING id, goal_id, user_id, description, done`,
		kr.Description, kr.Done, kr.ID, kr.UserID)
	var out models.KeyResult
	err := row.Scan(&out.ID, &out.GoalID, &out.UserID, &out.Description, &out.Done)
	return out, err
}

func (r *pgGoalRepo) DeleteKeyResult(ctx context.Context, krID, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM key_results WHERE id=$1 AND user_id=$2`, krID, userID)
	return err
}

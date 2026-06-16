package postgres

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/goals"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

func nullDateString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

func computeProgress(krs []goals.KeyResult) int {
	// Only one-time KRs count toward goal progress
	oneTime := make([]goals.KeyResult, 0, len(krs))
	for _, kr := range krs {
		if !kr.Recurring {
			oneTime = append(oneTime, kr)
		}
	}
	if len(oneTime) == 0 {
		return 0
	}
	done := 0
	for _, kr := range oneTime {
		if kr.Done {
			done++
		}
	}
	return int(float64(done) / float64(len(oneTime)) * 100)
}

type pgGoalRepo struct{ db *pgxpool.Pool }

func NewGoalRepo(db *pgxpool.Pool) repository.GoalRepo { return &pgGoalRepo{db} }

func (r *pgGoalRepo) List(ctx context.Context, userID string) ([]goals.Goal, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, name, description, target_date, progress, color, status, created_at
		 FROM goals WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var gs []goals.Goal
	for rows.Next() {
		var g goals.Goal
		var targetDate *time.Time
		rows.Scan(&g.ID, &g.UserID, &g.Name, &g.Description, &targetDate, &g.Progress, &g.Color, &g.Status, &g.CreatedAt)
		g.TargetDate = nullDateString(targetDate)
		gs = append(gs, g)
	}
	if gs == nil {
		gs = []goals.Goal{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i, g := range gs {
		krows, err := r.db.Query(ctx,
			`SELECT id, goal_id, user_id, description, done, recurring,
			        TO_CHAR(reminder_time, 'HH24:MI') AS reminder_time
			 FROM key_results WHERE goal_id = $1 ORDER BY created_at`, g.ID)
		if err != nil {
			return nil, err
		}
		var krs []goals.KeyResult
		for krows.Next() {
			var kr goals.KeyResult
			krows.Scan(&kr.ID, &kr.GoalID, &kr.UserID, &kr.Description, &kr.Done, &kr.Recurring, &kr.ReminderTime)
			krs = append(krs, kr)
		}
		krows.Close()
		if krs == nil {
			krs = []goals.KeyResult{}
		}
		gs[i].KeyResults = krs
		gs[i].Progress = computeProgress(krs)
	}
	return gs, nil
}

func (r *pgGoalRepo) Create(ctx context.Context, g goals.Goal) (goals.Goal, error) {
	if g.Status == "" {
		g.Status = "active"
	}
	row := r.db.QueryRow(ctx,
		`INSERT INTO goals (user_id, name, description, target_date, progress, color, status)
		 VALUES ($1, $2, $3, $4, 0, $5, $6)
		 RETURNING id, user_id, name, description, target_date, progress, color, status, created_at`,
		g.UserID, g.Name, g.Description, g.TargetDate, g.Color, g.Status)
	var out goals.Goal
	var targetDate *time.Time
	err := row.Scan(&out.ID, &out.UserID, &out.Name, &out.Description, &targetDate, &out.Progress, &out.Color, &out.Status, &out.CreatedAt)
	out.TargetDate = nullDateString(targetDate)
	out.KeyResults = []goals.KeyResult{}
	return out, err
}

func (r *pgGoalRepo) Update(ctx context.Context, g goals.Goal) (goals.Goal, error) {
	if g.Status == "" {
		g.Status = "active"
	}
	row := r.db.QueryRow(ctx,
		`UPDATE goals SET name=$1, description=$2, target_date=$3, color=$4, status=$5
		 WHERE id=$6 AND user_id=$7
		 RETURNING id, user_id, name, description, target_date, progress, color, status, created_at`,
		g.Name, g.Description, g.TargetDate, g.Color, g.Status, g.ID, g.UserID)
	var out goals.Goal
	var targetDate *time.Time
	err := row.Scan(&out.ID, &out.UserID, &out.Name, &out.Description, &targetDate, &out.Progress, &out.Color, &out.Status, &out.CreatedAt)
	out.TargetDate = nullDateString(targetDate)
	return out, err
}

func (r *pgGoalRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM goals WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

func (r *pgGoalRepo) AddKeyResult(ctx context.Context, kr goals.KeyResult) (goals.KeyResult, error) {
	var reminderArg interface{} = nil
	if kr.ReminderTime != nil && *kr.ReminderTime != "" {
		reminderArg = *kr.ReminderTime
	}
	row := r.db.QueryRow(ctx,
		`INSERT INTO key_results (goal_id, user_id, description, done, recurring, reminder_time)
		 VALUES ($1, $2, $3, FALSE, $4, $5::time)
		 RETURNING id, goal_id, user_id, description, done, recurring,
		           TO_CHAR(reminder_time, 'HH24:MI')`,
		kr.GoalID, kr.UserID, kr.Description, kr.Recurring, reminderArg)
	var out goals.KeyResult
	err := row.Scan(&out.ID, &out.GoalID, &out.UserID, &out.Description, &out.Done, &out.Recurring, &out.ReminderTime)
	return out, err
}

func (r *pgGoalRepo) UpdateKeyResult(ctx context.Context, kr goals.KeyResult) (goals.KeyResult, error) {
	var reminderArg interface{} = nil
	if kr.ReminderTime != nil && *kr.ReminderTime != "" {
		reminderArg = *kr.ReminderTime
	}
	row := r.db.QueryRow(ctx,
		`UPDATE key_results
		 SET description=$1, done=$2, recurring=$3, reminder_time=$4::time
		 WHERE id=$5 AND user_id=$6
		 RETURNING id, goal_id, user_id, description, done, recurring,
		           TO_CHAR(reminder_time, 'HH24:MI')`,
		kr.Description, kr.Done, kr.Recurring, reminderArg, kr.ID, kr.UserID)
	var out goals.KeyResult
	err := row.Scan(&out.ID, &out.GoalID, &out.UserID, &out.Description, &out.Done, &out.Recurring, &out.ReminderTime)
	return out, err
}

func (r *pgGoalRepo) DeleteKeyResult(ctx context.Context, krID, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM key_results WHERE id=$1 AND user_id=$2`, krID, userID)
	return err
}

func (r *pgGoalRepo) HabitsSummary(ctx context.Context, userID string) (total, doneToday int, err error) {
	row := r.db.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN kl.done THEN 1 ELSE 0 END), 0)
		 FROM key_results kr
		 LEFT JOIN kr_logs kl ON kl.kr_id = kr.id AND kl.logged_date = CURRENT_DATE
		 WHERE kr.user_id = $1 AND kr.recurring = TRUE`, userID)
	err = row.Scan(&total, &doneToday)
	return
}

func (r *pgGoalRepo) GoalsAvgProgress(ctx context.Context, userID string) (int, error) {
	var avg int
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(ROUND(AVG(
			CASE WHEN total = 0 THEN 0
			ELSE done_count::numeric / total * 100 END
		)), 0)
		FROM (
			SELECT g.id,
				COUNT(kr.id) AS total,
				COUNT(CASE WHEN kr.done THEN 1 END) AS done_count
			FROM goals g
			LEFT JOIN key_results kr ON kr.goal_id = g.id
			WHERE g.user_id = $1 AND g.status = 'active'
			GROUP BY g.id
		) sub`, userID).Scan(&avg)
	return avg, err
}

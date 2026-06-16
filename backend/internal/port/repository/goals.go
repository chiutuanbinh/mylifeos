package repository

import (
	"context"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/goals"
)

type GoalRepo interface {
	List(ctx context.Context, userID string) ([]goals.Goal, error)
	Create(ctx context.Context, g goals.Goal) (goals.Goal, error)
	Update(ctx context.Context, g goals.Goal) (goals.Goal, error)
	Delete(ctx context.Context, id, userID string) error
	AddKeyResult(ctx context.Context, kr goals.KeyResult) (goals.KeyResult, error)
	UpdateKeyResult(ctx context.Context, kr goals.KeyResult) (goals.KeyResult, error)
	DeleteKeyResult(ctx context.Context, krID, userID string) error
	HabitsSummary(ctx context.Context, userID string) (total, doneToday int, err error)
	GoalsAvgProgress(ctx context.Context, userID string) (int, error)
}

type KRLogRepo interface {
	GetLogs(ctx context.Context, userID, date string) ([]goals.KRLog, error)
	GetLogRange(ctx context.Context, krID, userID, from, to string) ([]goals.KRLog, error)
	ToggleLog(ctx context.Context, krID, userID, date string) (goals.KRLog, error)
}

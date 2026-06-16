package repository

import (
	"context"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/calendar"
)

type EventRepo interface {
	List(ctx context.Context, userID, from, to string) ([]calendar.Event, error)
	Create(ctx context.Context, e calendar.Event) (calendar.Event, error)
	Update(ctx context.Context, e calendar.Event) (calendar.Event, error)
	Delete(ctx context.Context, id, userID string) error
	UpsertFromGoogle(ctx context.Context, userID string, events []calendar.Event) (int, error)
}

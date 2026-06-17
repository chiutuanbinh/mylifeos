package repository

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
)

type AccountRepo interface {
	Save(ctx context.Context, a *accounting.Account) error
	FindByUser(ctx context.Context, userID string) ([]*accounting.Account, error)
	FindByID(ctx context.Context, id accounting.AccountID) (*accounting.Account, error)
}

type JournalRepo interface {
	Save(ctx context.Context, e *accounting.JournalEntry) error
	FindByUser(ctx context.Context, userID string, from, to time.Time) ([]*accounting.JournalEntry, error)
}

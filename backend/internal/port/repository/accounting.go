package repository

import (
	"context"
	"errors"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
)

var ErrAccountNotFound = errors.New("account not found")

type AccountRepo interface {
	Save(ctx context.Context, a *accounting.Account) error
	FindByUser(ctx context.Context, userID string) ([]*accounting.Account, error)
	FindByID(ctx context.Context, id accounting.AccountID, userID string) (*accounting.Account, error)
	FindByNameAndType(ctx context.Context, userID, name string, t accounting.AccountType) (*accounting.Account, error)
	Delete(ctx context.Context, id accounting.AccountID, userID string) error
}

type JournalRepo interface {
	Save(ctx context.Context, e *accounting.JournalEntry) error
	FindByUser(ctx context.Context, userID string, from, to time.Time) ([]*accounting.JournalEntry, error)
	SaveGoalLinks(ctx context.Context, entryID, userID string, goalIDs []string) error
}

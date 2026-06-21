package repository

import (
	"context"
	"errors"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/finance"
)

var ErrBudgetNotFound = errors.New("budget not found")

type TransactionRepo interface {
	ListBudgets(ctx context.Context, userID string) ([]finance.Budget, error)
	UpsertBudget(ctx context.Context, b finance.Budget) (finance.Budget, error)
	DeleteBudget(ctx context.Context, userID, category string) error
	SumByUser(ctx context.Context, userID string) (float64, error)
	SumSpentThisMonth(ctx context.Context, userID string) (float64, error)
}

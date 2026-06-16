package repository

import (
	"context"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/finance"
)

type TransactionRepo interface {
	List(ctx context.Context, userID, category, from, to string, limit, offset int) ([]finance.Transaction, error)
	Create(ctx context.Context, t finance.Transaction) (finance.Transaction, error)
	Delete(ctx context.Context, id, userID string) error
	ListBudgets(ctx context.Context, userID string) ([]finance.Budget, error)
	UpsertBudget(ctx context.Context, b finance.Budget) (finance.Budget, error)
	SumByUser(ctx context.Context, userID string) (float64, error)
	SumSpentThisMonth(ctx context.Context, userID string) (float64, error)
}

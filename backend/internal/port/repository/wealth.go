package repository

import (
	"context"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/wealth"
)

type AssetRepo interface {
	List(ctx context.Context, userID string) ([]wealth.Asset, error)
	Create(ctx context.Context, a wealth.Asset) (wealth.Asset, error)
	Update(ctx context.Context, a wealth.Asset) (wealth.Asset, error)
	Delete(ctx context.Context, id, userID string) error
}

type LiabilityRepo interface {
	List(ctx context.Context, userID string) ([]wealth.Liability, error)
	Create(ctx context.Context, l wealth.Liability) (wealth.Liability, error)
	Update(ctx context.Context, l wealth.Liability) (wealth.Liability, error)
	Delete(ctx context.Context, id, userID string) error
	TotalBalance(ctx context.Context, userID string) (float64, error)
}

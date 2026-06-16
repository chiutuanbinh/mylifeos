package repository

import (
	"context"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/settings"
)

type SettingsRepo interface {
	Get(ctx context.Context, userID string) (settings.UserSettings, error)
	Upsert(ctx context.Context, s settings.UserSettings) (settings.UserSettings, error)
}

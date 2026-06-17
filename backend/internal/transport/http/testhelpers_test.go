package httphandler_test

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
)

func withUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, middleware.UserIDKey, userID)
}

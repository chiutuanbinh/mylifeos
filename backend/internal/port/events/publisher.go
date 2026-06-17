package events

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
)

type Publisher interface {
	Publish(ctx context.Context, event accounting.DomainEvent) error
}

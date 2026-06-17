package infraevents

import (
	"context"
	"log"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/events"
)

type InProcessPublisher struct{}

func NewInProcessPublisher() events.Publisher {
	return &InProcessPublisher{}
}

func (p *InProcessPublisher) Publish(_ context.Context, ev accounting.DomainEvent) error {
	switch e := ev.(type) {
	case accounting.EntryPosted:
		log.Printf("accounting: entry posted userID=%s entryID=%s date=%s", e.UserID, e.EntryID, e.Date.Format("2006-01-02"))
	}
	return nil
}

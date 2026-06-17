package accountingsvc

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/events"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
)

type JournalService struct {
	journal repository.JournalRepo
	pub     events.Publisher
}

func NewJournalService(journal repository.JournalRepo, pub events.Publisher) *JournalService {
	return &JournalService{journal: journal, pub: pub}
}

func (s *JournalService) RecordTransaction(ctx context.Context, cmd RecordTransactionCmd) (accounting.EntryID, error) {
	entry := accounting.NewJournalEntry(cmd.UserID, cmd.Date, cmd.Description)
	entry.SetMemo(cmd.Memo)

	for _, l := range cmd.Lines {
		money, err := accounting.NewMoney(l.Amount, l.Currency)
		if err != nil {
			return "", err
		}
		if err := entry.AddLine(accounting.AccountID(l.AccountID), money, l.Side); err != nil {
			return "", err
		}
	}

	if err := entry.Post(); err != nil {
		return "", err
	}
	if err := s.journal.Save(ctx, entry); err != nil {
		return "", err
	}
	for _, ev := range entry.Events() {
		if err := s.pub.Publish(ctx, ev); err != nil {
			return "", err
		}
	}
	return entry.ID(), nil
}

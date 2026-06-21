package accountingsvc

import (
	"context"
	"fmt"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/events"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
)

type JournalService struct {
	journal  repository.JournalRepo
	accounts repository.AccountRepo
	pub      events.Publisher
}

func NewJournalService(journal repository.JournalRepo, accounts repository.AccountRepo, pub events.Publisher) *JournalService {
	return &JournalService{journal: journal, accounts: accounts, pub: pub}
}

func (s *JournalService) ListByUser(ctx context.Context, userID string) ([]*accounting.JournalEntry, error) {
	return s.journal.FindByUser(ctx, userID, time.Time{}, time.Now())
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

	// validate all line accounts belong to this user
	userAccounts, err := s.accounts.FindByUser(ctx, cmd.UserID)
	if err != nil {
		return "", err
	}
	owned := map[accounting.AccountID]bool{}
	for _, a := range userAccounts {
		owned[a.ID()] = true
	}
	for _, l := range cmd.Lines {
		if !owned[accounting.AccountID(l.AccountID)] {
			return "", fmt.Errorf("account %s does not belong to user", l.AccountID)
		}
	}

	// load full account objects for group check
	accountMap := map[accounting.AccountID]*accounting.Account{}
	for _, a := range userAccounts {
		accountMap[a.ID()] = a
	}
	for _, l := range cmd.Lines {
		acct := accountMap[accounting.AccountID(l.AccountID)]
		if acct.IsGroup() {
			return "", fmt.Errorf("account %s is a group account and cannot receive journal lines", l.AccountID)
		}
	}

	if err := entry.Post(); err != nil {
		return "", err
	}
	if err := s.journal.Save(ctx, entry); err != nil {
		return "", err
	}
	if len(cmd.GoalIDs) > 0 {
		if err := s.journal.SaveGoalLinks(ctx, string(entry.ID()), cmd.UserID, cmd.GoalIDs); err != nil {
			return "", err
		}
	}
	for _, ev := range entry.Events() {
		if err := s.pub.Publish(ctx, ev); err != nil {
			return "", err
		}
	}
	return entry.ID(), nil
}

package accountingsvc

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
)

type NetWorthQuery struct {
	accounts repository.AccountRepo
	journal  repository.JournalRepo
}

func NewNetWorthQuery(accounts repository.AccountRepo, journal repository.JournalRepo) *NetWorthQuery {
	return &NetWorthQuery{accounts: accounts, journal: journal}
}

func (q *NetWorthQuery) Current(ctx context.Context, userID string) (accounting.Money, error) {
	accounts, err := q.accounts.FindByUser(ctx, userID)
	if err != nil {
		return accounting.Money{}, err
	}
	entries, err := q.journal.FindByUser(ctx, userID, time.Time{}, time.Now())
	if err != nil {
		return accounting.Money{}, err
	}
	var lines []accounting.JournalLine
	for _, e := range entries {
		lines = append(lines, e.Lines()...)
	}
	return accounting.NetWorthService{}.Calculate(accounts, lines), nil
}

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

type NetWorthResult struct {
	NetWorth     accounting.Money
	NetIncomeYTD accounting.Money
}

func (q *NetWorthQuery) Current(ctx context.Context, userID string) (NetWorthResult, error) {
	accounts, err := q.accounts.FindByUser(ctx, userID)
	if err != nil {
		return NetWorthResult{}, err
	}
	allEntries, err := q.journal.FindByUser(ctx, userID, time.Time{}, time.Now())
	if err != nil {
		return NetWorthResult{}, err
	}
	var allLines []accounting.JournalLine
	for _, e := range allEntries {
		allLines = append(allLines, e.Lines()...)
	}

	nw, err := accounting.NetWorthService{}.Calculate(accounts, allLines)
	if err != nil {
		return NetWorthResult{}, err
	}

	// Net income YTD: income credits - expense debits since Jan 1
	ytdStart := time.Date(time.Now().Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	ytdEntries, err := q.journal.FindByUser(ctx, userID, ytdStart, time.Now())
	if err != nil {
		return NetWorthResult{}, err
	}

	// build account type index
	acctType := map[accounting.AccountID]accounting.AccountType{}
	for _, a := range accounts {
		acctType[a.ID()] = a.Type()
	}

	netIncome := accounting.ZeroMoney("VND")
	for _, e := range ytdEntries {
		for _, l := range e.Lines() {
			t := acctType[l.AccountID()]
			switch {
			case t == accounting.Income && l.Side() == accounting.Credit:
				netIncome, _ = netIncome.Add(accounting.Money{Amount: l.Money().Amount, Currency: l.Money().Currency})
			case t == accounting.Expense && l.Side() == accounting.Debit:
				netIncome = accounting.Money{Amount: netIncome.Amount.Sub(l.Money().Amount), Currency: netIncome.Currency}
			}
		}
	}

	return NetWorthResult{NetWorth: nw, NetIncomeYTD: netIncome}, nil
}

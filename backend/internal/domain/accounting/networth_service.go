package accounting

import (
	"fmt"

	"github.com/shopspring/decimal"
)

type NetWorthService struct{}

func (NetWorthService) Calculate(accounts []*Account, lines []JournalLine) (Money, error) {
	total := decimal.Zero
	for _, a := range accounts {
		if a.IsGroup() || a.Archived() {
			continue
		}
		bal, err := a.Balance(lines)
		if err != nil {
			return Money{}, err
		}
		if bal.Currency != "VND" {
			return Money{}, fmt.Errorf("multi-currency not supported: account %s has currency %s", a.Name(), bal.Currency)
		}
		switch a.Type() {
		case Asset:
			total = total.Add(bal.Amount)
		case Liability:
			total = total.Sub(bal.Amount)
		}
	}
	return Money{Amount: total, Currency: "VND"}, nil
}

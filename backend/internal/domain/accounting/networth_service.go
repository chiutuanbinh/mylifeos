package accounting

import "github.com/shopspring/decimal"

type NetWorthService struct{}

func (NetWorthService) Calculate(accounts []*Account, lines []JournalLine) Money {
	total := decimal.Zero
	for _, a := range accounts {
		if a.IsGroup() || a.Archived() {
			continue
		}
		bal := a.Balance(lines)
		switch a.Type() {
		case Asset:
			total = total.Add(bal.Amount)
		case Liability:
			total = total.Sub(bal.Amount)
		}
	}
	return Money{Amount: total, Currency: "VND"}
}

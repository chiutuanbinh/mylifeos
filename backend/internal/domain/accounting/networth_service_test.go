package accounting_test

import (
	"testing"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/shopspring/decimal"
)

func TestNetWorthService_Calculate(t *testing.T) {
	cash    := accounting.NewAccount("u1", nil, "Cash",    accounting.Asset,     "VND", false, 0)
	visa    := accounting.NewAccount("u1", nil, "Visa",    accounting.Liability, "VND", false, 0)
	salary  := accounting.NewAccount("u1", nil, "Salary",  accounting.Income,    "VND", false, 0)
	food    := accounting.NewAccount("u1", nil, "Food",    accounting.Expense,   "VND", false, 0)

	// Salary received: debit Cash 10M, credit Salary 10M
	e1 := accounting.NewJournalEntry("u1", time.Now(), "Salary")
	e1.AddLine(cash.ID(),   mustMoney(10_000_000, "VND"), accounting.Debit)
	e1.AddLine(salary.ID(), mustMoney(10_000_000, "VND"), accounting.Credit)
	e1.Post()

	// Buy with Visa: debit Food 150k, credit Visa 150k
	e2 := accounting.NewJournalEntry("u1", time.Now(), "Coffee")
	e2.AddLine(food.ID(), mustMoney(150_000, "VND"), accounting.Debit)
	e2.AddLine(visa.ID(), mustMoney(150_000, "VND"), accounting.Credit)
	e2.Post()

	var lines []accounting.JournalLine
	lines = append(lines, e1.Lines()...)
	lines = append(lines, e2.Lines()...)

	svc := accounting.NetWorthService{}
	nw := svc.Calculate([]*accounting.Account{cash, visa, salary, food}, lines)

	// Cash = 10M (asset), Visa = 150k (liability)
	// Net worth = 10M - 150k = 9,850,000
	want := decimal.NewFromInt(9_850_000)
	if !nw.Amount.Equal(want) {
		t.Errorf("want %s, got %s", want, nw.Amount)
	}
}

func TestNetWorthService_SkipsGroupAccounts(t *testing.T) {
	group := accounting.NewAccount("u1", nil, "Assets", accounting.Asset, "VND", true, 0)
	leaf  := accounting.NewAccount("u1", nil, "Cash",   accounting.Asset, "VND", false, 0)

	e := accounting.NewJournalEntry("u1", time.Now(), "Test")
	e.AddLine(leaf.ID(),  mustMoney(100, "VND"), accounting.Debit)
	e.AddLine(group.ID(), mustMoney(100, "VND"), accounting.Credit) // unusual but tests skip logic
	e.Post()

	svc := accounting.NetWorthService{}
	nw := svc.Calculate([]*accounting.Account{group, leaf}, e.Lines())

	// group skipped, leaf = 100 asset
	if !nw.Amount.Equal(decimal.NewFromInt(100)) {
		t.Errorf("want 100, got %s", nw.Amount)
	}
}

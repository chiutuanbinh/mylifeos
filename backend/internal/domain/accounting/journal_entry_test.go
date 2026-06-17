package accounting_test

import (
	"testing"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/shopspring/decimal"
)

func mustMoney(amount int64, currency string) accounting.Money {
	m, _ := accounting.NewMoney(decimal.NewFromInt(amount), currency)
	return m
}

func TestJournalEntry_Post_Balanced(t *testing.T) {
	e := accounting.NewJournalEntry("user1", time.Now(), "Coffee")
	e.AddLine("account-expense", mustMoney(150000, "VND"), accounting.Debit)
	e.AddLine("account-visa", mustMoney(150000, "VND"), accounting.Credit)
	if err := e.Post(); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
}

func TestJournalEntry_Post_Unbalanced(t *testing.T) {
	e := accounting.NewJournalEntry("user1", time.Now(), "Bad entry")
	e.AddLine("account-a", mustMoney(100, "VND"), accounting.Debit)
	e.AddLine("account-b", mustMoney(50, "VND"), accounting.Credit)
	if err := e.Post(); err == nil {
		t.Fatal("want error for unbalanced entry")
	}
}

func TestJournalEntry_Post_TooFewLines(t *testing.T) {
	e := accounting.NewJournalEntry("user1", time.Now(), "One line")
	e.AddLine("account-a", mustMoney(100, "VND"), accounting.Debit)
	if err := e.Post(); err == nil {
		t.Fatal("want error for single line")
	}
}

func TestJournalEntry_Post_EmitsEvent(t *testing.T) {
	e := accounting.NewJournalEntry("user1", time.Now(), "Coffee")
	e.AddLine("account-expense", mustMoney(100, "VND"), accounting.Debit)
	e.AddLine("account-visa", mustMoney(100, "VND"), accounting.Credit)
	e.Post()
	evs := e.Events()
	if len(evs) != 1 {
		t.Fatalf("want 1 event, got %d", len(evs))
	}
	ep, ok := evs[0].(accounting.EntryPosted)
	if !ok {
		t.Fatal("want EntryPosted event")
	}
	if ep.UserID != "user1" {
		t.Errorf("want userID user1, got %s", ep.UserID)
	}
}

func TestJournalEntry_AddLine_ZeroAmountRejected(t *testing.T) {
	e := accounting.NewJournalEntry("user1", time.Now(), "Zero")
	m, _ := accounting.NewMoney(decimal.Zero, "VND")
	if err := e.AddLine("account-a", m, accounting.Debit); err == nil {
		t.Fatal("want error for zero amount line")
	}
}

func TestJournalEntry_Lines_DefensiveCopy(t *testing.T) {
	e := accounting.NewJournalEntry("user1", time.Now(), "Test")
	e.AddLine("a", mustMoney(100, "VND"), accounting.Debit)
	e.AddLine("b", mustMoney(100, "VND"), accounting.Credit)
	lines := e.Lines()
	lines[0] = accounting.JournalLine{}
	if e.Lines()[0].Money().Amount.IsZero() {
		t.Error("Lines() should return a defensive copy")
	}
}

func TestAccount_Balance_WithRealLines(t *testing.T) {
	a := accounting.NewAccount("user1", nil, "Cash", accounting.Asset, "VND", false, 0)
	entry := accounting.NewJournalEntry("user1", time.Now(), "Test")
	entry.AddLine(a.ID(), mustMoney(500, "VND"), accounting.Debit)
	entry.AddLine("other", mustMoney(500, "VND"), accounting.Credit)
	entry.Post()

	bal, err := a.Balance(entry.Lines())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bal.Amount.Equal(decimal.NewFromInt(500)) {
		t.Errorf("want 500, got %s", bal.Amount)
	}
}

package accounting_test

import (
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/shopspring/decimal"
)

func TestNewMoney_RejectsNegative(t *testing.T) {
	_, err := accounting.NewMoney(decimal.NewFromInt(-1), "VND")
	if err == nil {
		t.Fatal("want error for negative amount")
	}
}

func TestNewMoney_AcceptsZero(t *testing.T) {
	m, err := accounting.NewMoney(decimal.Zero, "VND")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.Amount.IsZero() {
		t.Error("want zero amount")
	}
}

func TestMoney_Add_SameCurrency(t *testing.T) {
	a, _ := accounting.NewMoney(decimal.NewFromInt(100), "VND")
	b, _ := accounting.NewMoney(decimal.NewFromInt(50), "VND")
	got, err := a.Add(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Amount.Equal(decimal.NewFromInt(150)) {
		t.Errorf("want 150, got %s", got.Amount)
	}
}

func TestMoney_Add_CurrencyMismatch(t *testing.T) {
	a, _ := accounting.NewMoney(decimal.NewFromInt(100), "VND")
	b, _ := accounting.NewMoney(decimal.NewFromInt(50), "USD")
	_, err := a.Add(b)
	if err == nil {
		t.Fatal("want error for currency mismatch")
	}
}

package accounting_test

import (
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
)

func TestAccount_NormalBalance(t *testing.T) {
	cases := []struct {
		t    accounting.AccountType
		want accounting.Side
	}{
		{accounting.Asset, accounting.Debit},
		{accounting.Expense, accounting.Debit},
		{accounting.Liability, accounting.Credit},
		{accounting.Equity, accounting.Credit},
		{accounting.Income, accounting.Credit},
	}
	for _, c := range cases {
		a := accounting.NewAccount("user1", nil, "Test", c.t, "VND", false, 0)
		if a.NormalBalance() != c.want {
			t.Errorf("type %s: want %s, got %s", c.t, c.want, a.NormalBalance())
		}
	}
}

func TestNewAccount_IDNotEmpty(t *testing.T) {
	a := accounting.NewAccount("user1", nil, "Cash", accounting.Asset, "VND", false, 0)
	if string(a.ID()) == "" {
		t.Error("want non-empty ID")
	}
}

func TestNewAccount_ParentID_Nil(t *testing.T) {
	a := accounting.NewAccount("user1", nil, "Cash", accounting.Asset, "VND", false, 0)
	if a.ParentID() != nil {
		t.Error("want nil ParentID")
	}
}

func TestNewAccount_ParentID_Set(t *testing.T) {
	pid := "parent-123"
	a := accounting.NewAccount("user1", &pid, "Cash", accounting.Asset, "VND", false, 0)
	if a.ParentID() == nil || string(*a.ParentID()) != pid {
		t.Error("want ParentID set")
	}
}

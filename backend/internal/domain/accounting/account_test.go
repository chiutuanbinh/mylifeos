package accounting_test

import (
	"testing"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/shopspring/decimal"
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

func TestAccount_Rename(t *testing.T) {
	a := accounting.NewAccount("u1", nil, "Old", accounting.Asset, "VND", false, 0)
	a.Rename("New")
	if a.Name() != "New" {
		t.Errorf("want Name=New, got %s", a.Name())
	}
}

func TestAccount_ChangeType(t *testing.T) {
	a := accounting.NewAccount("u1", nil, "X", accounting.Asset, "VND", false, 0)
	a.ChangeType(accounting.Expense)
	if a.Type() != accounting.Expense {
		t.Errorf("want Expense, got %s", a.Type())
	}
}

func TestAccount_Reparent(t *testing.T) {
	pid := accounting.AccountID("parent-1")
	a := accounting.NewAccount("u1", nil, "X", accounting.Asset, "VND", false, 0)
	a.Reparent(&pid)
	if a.ParentID() == nil || *a.ParentID() != pid {
		t.Error("want ParentID set")
	}
	a.Reparent(nil)
	if a.ParentID() != nil {
		t.Error("want ParentID nil after clear")
	}
}

func TestAccount_Reorder(t *testing.T) {
	a := accounting.NewAccount("u1", nil, "X", accounting.Asset, "VND", false, 0)
	a.Reorder(5)
	if a.SortOrder() != 5 {
		t.Errorf("want SortOrder=5, got %d", a.SortOrder())
	}
}

func TestAccount_AttachAssetMeta(t *testing.T) {
	a := accounting.NewAccount("u1", nil, "Car", accounting.Asset, "VND", false, 0)
	if a.AssetMeta() != nil {
		t.Error("want nil AssetMeta initially")
	}
	pv := decimal.NewFromInt(500_000_000)
	now := time.Now()
	dr := decimal.NewFromFloat(0.15)
	a.AttachAssetMeta(&accounting.AssetMeta{
		PurchaseValue:    &pv,
		PurchasedAt:      &now,
		DepreciationRate: &dr,
		Notes:            "Toyota",
	})
	if a.AssetMeta() == nil {
		t.Fatal("want non-nil AssetMeta")
	}
	if a.AssetMeta().Notes != "Toyota" {
		t.Errorf("want Notes=Toyota, got %s", a.AssetMeta().Notes)
	}
}

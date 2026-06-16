package wealthsvc_test

import (
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/wealth"
	wealthsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/wealth"
)

func TestCurrentValue_NoDepreciation(t *testing.T) {
	pv := 100.0
	a := wealth.Asset{Value: 80, PurchaseValue: &pv, DepreciationRate: 0}
	got := wealthsvc.CurrentValue(a)
	if got != 100.0 {
		t.Errorf("want 100, got %v", got)
	}
}

func TestCurrentValue_FallbackToValue(t *testing.T) {
	a := wealth.Asset{Value: 80}
	got := wealthsvc.CurrentValue(a)
	if got != 80.0 {
		t.Errorf("want 80, got %v", got)
	}
}

func TestNetWorth(t *testing.T) {
	liabilities := []wealth.Liability{
		{Balance: 30},
		{Balance: 20},
	}
	got := wealthsvc.NetWorth(150, 50, liabilities)
	// 150 + 50 - 50 = 150
	if got != 150 {
		t.Errorf("want 150, got %v", got)
	}
}

func TestNetWorth_NoLiabilities(t *testing.T) {
	got := wealthsvc.NetWorth(100, 25, nil)
	if got != 125 {
		t.Errorf("want 125, got %v", got)
	}
}

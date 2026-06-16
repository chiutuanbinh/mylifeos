package wealthsvc

import (
	"math"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/wealth"
)

// CurrentValue returns the depreciation-adjusted value of an asset.
// If no PurchaseValue is set, falls back to Value.
func CurrentValue(a wealth.Asset) float64 {
	if a.PurchaseValue == nil || *a.PurchaseValue == 0 {
		return a.Value
	}
	if a.PurchasedAt == nil || a.DepreciationRate == 0 {
		return *a.PurchaseValue
	}
	t, err := time.Parse("2006-01-02", *a.PurchasedAt)
	if err != nil {
		return *a.PurchaseValue
	}
	years := time.Since(t).Hours() / 8760
	return *a.PurchaseValue * math.Pow(1-a.DepreciationRate, years)
}

// NetWorth computes net worth from pre-summed asset value, cash position, and liabilities.
func NetWorth(assetsTotal, cashPosition float64, liabilities []wealth.Liability) float64 {
	var totalLiabilities float64
	for _, l := range liabilities {
		totalLiabilities += l.Balance
	}
	return assetsTotal + cashPosition - totalLiabilities
}

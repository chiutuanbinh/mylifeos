package accountingsvc

import (
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/shopspring/decimal"
)

type LineCmd struct {
	AccountID string
	Amount    decimal.Decimal
	Currency  string
	Side      accounting.Side
}

type RecordTransactionCmd struct {
	UserID      string
	Date        time.Time
	Description string
	Memo        string
	Lines       []LineCmd
}

type AssetMetaCmd struct {
	PurchaseValue    *decimal.Decimal
	PurchasedAt      *time.Time
	DepreciationRate *decimal.Decimal
	Notes            string
}

type OpenAccountCmd struct {
	UserID         string
	ParentID       *string
	Name           string
	Type           accounting.AccountType
	Currency       string
	IsGroup        bool
	SortOrder      int
	OpeningBalance *decimal.Decimal
	AssetMeta      *AssetMetaCmd
}

type UpdateAccountCmd struct {
	ID        string
	UserID    string
	Name      string
	Type      accounting.AccountType
	ParentID  *string
	SortOrder int
	AssetMeta *AssetMetaCmd
}

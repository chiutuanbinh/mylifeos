package accounting

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type AccountID string
type AccountType string
type Side string

const (
	Asset     AccountType = "asset"
	Liability AccountType = "liability"
	Equity    AccountType = "equity"
	Income    AccountType = "income"
	Expense   AccountType = "expense"
)

const (
	Debit  Side = "debit"
	Credit Side = "credit"
)

type AssetMeta struct {
	PurchaseValue    *decimal.Decimal
	PurchasedAt      *time.Time
	DepreciationRate *decimal.Decimal
	Notes            string
}

type Account struct {
	id        AccountID
	userID    string
	parentID  *AccountID
	name      string
	acctType  AccountType
	currency  string
	isGroup   bool
	archived  bool
	sortOrder int
	assetMeta *AssetMeta
}

func NewAccount(userID string, parentID *string, name string, acctType AccountType, currency string, isGroup bool, sortOrder int) *Account {
	var pid *AccountID
	if parentID != nil {
		p := AccountID(*parentID)
		pid = &p
	}
	return &Account{
		id:        AccountID(newID()),
		userID:    userID,
		parentID:  pid,
		name:      name,
		acctType:  acctType,
		currency:  currency,
		isGroup:   isGroup,
		sortOrder: sortOrder,
	}
}

func ReconstituteAccount(id, userID string, parentID *string, name string, acctType AccountType, currency string, isGroup, archived bool, sortOrder int) *Account {
	var pid *AccountID
	if parentID != nil {
		p := AccountID(*parentID)
		pid = &p
	}
	return &Account{
		id:        AccountID(id),
		userID:    userID,
		parentID:  pid,
		name:      name,
		acctType:  acctType,
		currency:  currency,
		isGroup:   isGroup,
		archived:  archived,
		sortOrder: sortOrder,
	}
}

func (a *Account) ID() AccountID        { return a.id }
func (a *Account) UserID() string        { return a.userID }
func (a *Account) ParentID() *AccountID  { return a.parentID }
func (a *Account) Name() string          { return a.name }
func (a *Account) Type() AccountType     { return a.acctType }
func (a *Account) Currency() string      { return a.currency }
func (a *Account) IsGroup() bool         { return a.isGroup }
func (a *Account) Archived() bool        { return a.archived }
func (a *Account) SortOrder() int        { return a.sortOrder }
func (a *Account) AssetMeta() *AssetMeta { return a.assetMeta }

// Mutation methods
func (a *Account) Rename(name string)           { a.name = name }
func (a *Account) ChangeType(t AccountType)     { a.acctType = t }
func (a *Account) Reparent(parentID *AccountID) { a.parentID = parentID }
func (a *Account) Reorder(n int)                { a.sortOrder = n }
func (a *Account) AttachAssetMeta(m *AssetMeta) { a.assetMeta = m }

func (a *Account) NormalBalance() Side {
	switch a.acctType {
	case Asset, Expense:
		return Debit
	default:
		return Credit
	}
}

func (a *Account) Balance(lines []JournalLine) (Money, error) {
	normal := a.NormalBalance()
	total := Money{Amount: zeroDecimal(), Currency: a.currency}
	for _, l := range lines {
		if l.AccountID() != a.id {
			continue
		}
		lineAmount := Money{Amount: l.Money().Amount, Currency: l.Money().Currency}
		var err error
		if l.Side() == normal {
			total, err = total.Add(lineAmount)
		} else {
			if total.Currency != lineAmount.Currency {
				return Money{}, fmt.Errorf("currency mismatch in account %s: %s vs %s", a.id, total.Currency, lineAmount.Currency)
			}
			total = Money{Amount: total.Amount.Sub(lineAmount.Amount), Currency: total.Currency}
		}
		if err != nil {
			return Money{}, fmt.Errorf("currency mismatch in account %s: %w", a.id, err)
		}
	}
	return total, nil
}

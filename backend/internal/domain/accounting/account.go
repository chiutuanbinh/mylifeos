package accounting

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

func ReconstitueAccount(id, userID string, parentID *string, name string, acctType AccountType, currency string, isGroup, archived bool, sortOrder int) *Account {
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

func (a *Account) ID() AccountID       { return a.id }
func (a *Account) UserID() string       { return a.userID }
func (a *Account) ParentID() *AccountID { return a.parentID }
func (a *Account) Name() string         { return a.name }
func (a *Account) Type() AccountType    { return a.acctType }
func (a *Account) Currency() string     { return a.currency }
func (a *Account) IsGroup() bool        { return a.isGroup }
func (a *Account) Archived() bool       { return a.archived }
func (a *Account) SortOrder() int       { return a.sortOrder }

func (a *Account) NormalBalance() Side {
	switch a.acctType {
	case Asset, Expense:
		return Debit
	default:
		return Credit
	}
}

func (a *Account) Balance(lines []JournalLine) Money {
	normal := a.NormalBalance()
	total := zeroDecimal()
	for _, l := range lines {
		if l.AccountID() != a.id {
			continue
		}
		if l.Side() == normal {
			total = total.Add(l.Money().Amount)
		} else {
			total = total.Sub(l.Money().Amount)
		}
	}
	return Money{Amount: total, Currency: a.currency}
}

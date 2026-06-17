package accounting

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type EntryID string

type JournalLine struct {
	id        string
	accountID AccountID
	money     Money
	side      Side
}

func (l JournalLine) ID() string           { return l.id }
func (l JournalLine) AccountID() AccountID { return l.accountID }
func (l JournalLine) Money() Money         { return l.money }
func (l JournalLine) Side() Side           { return l.side }

type JournalEntry struct {
	id          EntryID
	userID      string
	date        time.Time
	description string
	memo        string
	lines       []JournalLine
	events      []DomainEvent
}

func NewJournalEntry(userID string, date time.Time, description string) *JournalEntry {
	return &JournalEntry{
		id:          EntryID(uuid.New().String()),
		userID:      userID,
		date:        date,
		description: description,
	}
}

func ReconstituteEntry(id, userID string, date time.Time, desc, memo string) *JournalEntry {
	return &JournalEntry{
		id:          EntryID(id),
		userID:      userID,
		date:        date,
		description: desc,
		memo:        memo,
	}
}

func (e *JournalEntry) SetMemo(memo string) { e.memo = memo }

func (e *JournalEntry) AddLine(accountID AccountID, money Money, side Side) error {
	if money.Amount.IsZero() {
		return errors.New("line amount must be non-zero")
	}
	e.lines = append(e.lines, JournalLine{
		id:        uuid.New().String(),
		accountID: accountID,
		money:     money,
		side:      side,
	})
	return nil
}

func (e *JournalEntry) ReconstituteLine(id string, acctID AccountID, money Money, side Side) {
	e.lines = append(e.lines, JournalLine{id: id, accountID: acctID, money: money, side: side})
}

func (e *JournalEntry) Post() error {
	if len(e.lines) < 2 {
		return errors.New("journal entry must have at least 2 lines")
	}

	// validate balance per currency
	type currencyBalance struct {
		debits  decimal.Decimal
		credits decimal.Decimal
	}
	byCurrency := map[string]*currencyBalance{}
	for _, l := range e.lines {
		if l.money.Amount.IsZero() {
			return errors.New("journal line amount must be non-zero")
		}
		cur := l.money.Currency
		if byCurrency[cur] == nil {
			byCurrency[cur] = &currencyBalance{}
		}
		if l.side == Debit {
			byCurrency[cur].debits = byCurrency[cur].debits.Add(l.money.Amount)
		} else {
			byCurrency[cur].credits = byCurrency[cur].credits.Add(l.money.Amount)
		}
	}
	for cur, bal := range byCurrency {
		if !bal.debits.Equal(bal.credits) {
			return fmt.Errorf("journal entry does not balance for currency %s: debits %s != credits %s",
				cur, bal.debits.String(), bal.credits.String())
		}
	}

	e.events = append(e.events, EntryPosted{
		EntryID: e.id,
		UserID:  e.userID,
		Date:    e.date,
	})
	return nil
}

func (e *JournalEntry) ID() EntryID          { return e.id }
func (e *JournalEntry) UserID() string        { return e.userID }
func (e *JournalEntry) Date() time.Time       { return e.date }
func (e *JournalEntry) Description() string   { return e.description }
func (e *JournalEntry) Memo() string          { return e.memo }
func (e *JournalEntry) Lines() []JournalLine  { return slices.Clone(e.lines) }
func (e *JournalEntry) Events() []DomainEvent { return slices.Clone(e.events) }

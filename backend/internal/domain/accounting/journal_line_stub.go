package accounting

// JournalLine stub — replaced by full implementation in Task 3.
// Delete this file when journal_entry.go is created.
type JournalLine struct {
	accountID AccountID
	money     Money
	side      Side
}

func (l JournalLine) AccountID() AccountID { return l.accountID }
func (l JournalLine) Money() Money         { return l.money }
func (l JournalLine) Side() Side           { return l.side }

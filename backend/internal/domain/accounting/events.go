package accounting

import "time"

type DomainEvent interface{ domainEvent() }

type EntryPosted struct {
	EntryID EntryID
	UserID  string
	Date    time.Time
}

func (EntryPosted) domainEvent() {}

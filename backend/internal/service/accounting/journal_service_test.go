package accountingsvc_test

import (
	"context"
	"testing"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
	"github.com/shopspring/decimal"
)

// --- fakes ---

type fakeJournalRepo struct {
	saved []*accounting.JournalEntry
}

func (r *fakeJournalRepo) Save(_ context.Context, e *accounting.JournalEntry) error {
	r.saved = append(r.saved, e)
	return nil
}

func (r *fakeJournalRepo) FindByUser(_ context.Context, _ string, _, _ time.Time) ([]*accounting.JournalEntry, error) {
	return r.saved, nil
}

type fakePublisher struct {
	published []accounting.DomainEvent
}

func (p *fakePublisher) Publish(_ context.Context, ev accounting.DomainEvent) error {
	p.published = append(p.published, ev)
	return nil
}

// newFakeAccountRepoWithIDs builds a fakeAccountRepo pre-populated with accounts
// owned by userID and having the given IDs.
func newFakeAccountRepoWithIDs(userID string, ids ...string) *fakeAccountRepo {
	r := &fakeAccountRepo{accounts: map[accounting.AccountID]*accounting.Account{}}
	for _, id := range ids {
		a := accounting.ReconstitueAccount(id, userID, nil, id, accounting.Asset, "VND", false, false, 0)
		r.accounts[a.ID()] = a
	}
	return r
}

// --- tests ---

func TestJournalService_RecordTransaction_Balanced(t *testing.T) {
	repo := &fakeJournalRepo{}
	pub := &fakePublisher{}
	ar := newFakeAccountRepoWithIDs("user1", "account-food", "account-visa")
	svc := accountingsvc.NewJournalService(repo, ar, pub)

	cmd := accountingsvc.RecordTransactionCmd{
		UserID:      "user1",
		Date:        time.Now(),
		Description: "Coffee",
		Lines: []accountingsvc.LineCmd{
			{AccountID: "account-food", Amount: decimal.NewFromInt(150000), Currency: "VND", Side: accounting.Debit},
			{AccountID: "account-visa", Amount: decimal.NewFromInt(150000), Currency: "VND", Side: accounting.Credit},
		},
	}
	id, err := svc.RecordTransaction(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(id) == "" {
		t.Error("want non-empty entry ID")
	}
	if len(repo.saved) != 1 {
		t.Error("want 1 saved entry")
	}
	if len(pub.published) != 1 {
		t.Error("want 1 published event")
	}
}

func TestJournalService_RecordTransaction_UnbalancedReturnsError(t *testing.T) {
	repo := &fakeJournalRepo{}
	pub := &fakePublisher{}
	ar := newFakeAccountRepoWithIDs("user1", "a", "b")
	svc := accountingsvc.NewJournalService(repo, ar, pub)

	cmd := accountingsvc.RecordTransactionCmd{
		UserID:      "user1",
		Date:        time.Now(),
		Description: "Bad",
		Lines: []accountingsvc.LineCmd{
			{AccountID: "a", Amount: decimal.NewFromInt(100), Currency: "VND", Side: accounting.Debit},
			{AccountID: "b", Amount: decimal.NewFromInt(50), Currency: "VND", Side: accounting.Credit},
		},
	}
	_, err := svc.RecordTransaction(context.Background(), cmd)
	if err == nil {
		t.Fatal("want error for unbalanced entry")
	}
	if len(repo.saved) != 0 {
		t.Error("unbalanced entry must not be saved")
	}
}

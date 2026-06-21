package accountingsvc_test

import (
	"context"
	"testing"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
	"github.com/shopspring/decimal"
)

type fakeAccountRepo struct {
	accounts map[accounting.AccountID]*accounting.Account
}

func newFakeAccountRepo() *fakeAccountRepo {
	return &fakeAccountRepo{accounts: map[accounting.AccountID]*accounting.Account{}}
}

func (r *fakeAccountRepo) Save(_ context.Context, a *accounting.Account) error {
	r.accounts[a.ID()] = a
	return nil
}

func (r *fakeAccountRepo) FindByUser(_ context.Context, userID string) ([]*accounting.Account, error) {
	var result []*accounting.Account
	for _, a := range r.accounts {
		if a.UserID() == userID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (r *fakeAccountRepo) FindByID(_ context.Context, id accounting.AccountID) (*accounting.Account, error) {
	a, ok := r.accounts[id]
	if !ok {
		return nil, repository.ErrAccountNotFound
	}
	return a, nil
}

func (r *fakeAccountRepo) FindByNameAndType(_ context.Context, userID, name string, t accounting.AccountType) (*accounting.Account, error) {
	for _, a := range r.accounts {
		if a.UserID() == userID && a.Name() == name && a.Type() == t {
			return a, nil
		}
	}
	return nil, repository.ErrAccountNotFound
}

func (r *fakeAccountRepo) Delete(_ context.Context, id accounting.AccountID) error {
	delete(r.accounts, id)
	return nil
}

type memJournalRepo struct {
	entries []*accounting.JournalEntry
}

func newMemJournalRepo() *memJournalRepo { return &memJournalRepo{} }

func (r *memJournalRepo) Save(_ context.Context, e *accounting.JournalEntry) error {
	r.entries = append(r.entries, e)
	return nil
}

func (r *memJournalRepo) FindByUser(_ context.Context, userID string, from, to time.Time) ([]*accounting.JournalEntry, error) {
	var res []*accounting.JournalEntry
	for _, e := range r.entries {
		if e.UserID() == userID {
			res = append(res, e)
		}
	}
	return res, nil
}

func TestAccountService_OpenAccount_Root(t *testing.T) {
	repo := newFakeAccountRepo()
	svc := accountingsvc.NewAccountService(repo, nil)

	cmd := accountingsvc.OpenAccountCmd{
		UserID:   "user1",
		Name:     "Cash",
		Type:     accounting.Asset,
		Currency: "VND",
		IsGroup:  false,
	}
	id, err := svc.OpenAccount(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(id) == "" {
		t.Error("want non-empty ID")
	}
}

func TestAccountService_OpenAccount_ParentMustBeGroup(t *testing.T) {
	repo := newFakeAccountRepo()
	svc := accountingsvc.NewAccountService(repo, nil)

	// Create a leaf account (isGroup=false)
	leaf := accounting.NewAccount("user1", nil, "Leaf", accounting.Asset, "VND", false, 0)
	repo.Save(context.Background(), leaf)

	pid := string(leaf.ID())
	cmd := accountingsvc.OpenAccountCmd{
		UserID:   "user1",
		ParentID: &pid,
		Name:     "Child",
		Type:     accounting.Asset,
		Currency: "VND",
	}
	_, err := svc.OpenAccount(context.Background(), cmd)
	if err == nil {
		t.Fatal("want error: parent is not a group")
	}
}

func TestAccountService_ListAccounts(t *testing.T) {
	repo := newFakeAccountRepo()
	svc := accountingsvc.NewAccountService(repo, nil)

	svc.OpenAccount(context.Background(), accountingsvc.OpenAccountCmd{UserID: "user1", Name: "Cash", Type: accounting.Asset, Currency: "VND"})
	svc.OpenAccount(context.Background(), accountingsvc.OpenAccountCmd{UserID: "user1", Name: "Bank", Type: accounting.Asset, Currency: "VND"})
	svc.OpenAccount(context.Background(), accountingsvc.OpenAccountCmd{UserID: "user2", Name: "Other", Type: accounting.Asset, Currency: "VND"})

	list, err := svc.ListAccounts(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("want 2 accounts for user1, got %d", len(list))
	}
}

func TestAccountService_UpdateAccount(t *testing.T) {
	repo := newFakeAccountRepo()
	svc := accountingsvc.NewAccountService(repo, nil)

	id, err := svc.OpenAccount(context.Background(), accountingsvc.OpenAccountCmd{
		UserID: "u1", Name: "Old Name", Type: accounting.Asset, Currency: "VND",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = svc.UpdateAccount(context.Background(), accountingsvc.UpdateAccountCmd{
		ID: string(id), UserID: "u1", Name: "New Name", Type: accounting.Expense, SortOrder: 3,
	})
	if err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}

	accounts, _ := svc.ListAccounts(context.Background(), "u1")
	if len(accounts) != 1 || accounts[0].Name() != "New Name" || accounts[0].Type() != accounting.Expense {
		t.Errorf("want updated account, got %+v", accounts)
	}
}

func TestAccountService_UpdateAccount_WrongUser(t *testing.T) {
	repo := newFakeAccountRepo()
	svc := accountingsvc.NewAccountService(repo, nil)

	id, _ := svc.OpenAccount(context.Background(), accountingsvc.OpenAccountCmd{
		UserID: "u1", Name: "X", Type: accounting.Asset, Currency: "VND",
	})

	err := svc.UpdateAccount(context.Background(), accountingsvc.UpdateAccountCmd{
		ID: string(id), UserID: "u2", Name: "Hacked",
	})
	if err == nil {
		t.Error("want error for wrong user")
	}
}

func TestAccountService_OpenAccount_WithOpeningBalance(t *testing.T) {
	repo := newFakeAccountRepo()
	journalRepo := newMemJournalRepo()
	svc := accountingsvc.NewAccountService(repo, journalRepo)

	_, err := svc.OpenAccount(context.Background(), accountingsvc.OpenAccountCmd{
		UserID: "u1", Name: "Opening Balance", Type: accounting.Equity, Currency: "VND", IsGroup: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	ob := decimal.NewFromInt(1_000_000)
	_, err = svc.OpenAccount(context.Background(), accountingsvc.OpenAccountCmd{
		UserID: "u1", Name: "Cash", Type: accounting.Asset, Currency: "VND",
		OpeningBalance: &ob,
	})
	if err != nil {
		t.Fatalf("OpenAccount with opening balance: %v", err)
	}

	entries, _ := journalRepo.FindByUser(context.Background(), "u1", time.Time{}, time.Now())
	if len(entries) != 1 {
		t.Fatalf("want 1 journal entry, got %d", len(entries))
	}
	if len(entries[0].Lines()) != 2 {
		t.Errorf("want 2 lines, got %d", len(entries[0].Lines()))
	}
}

func TestAccountService_OpenAccount_OpeningBalance_NoEquityAccount(t *testing.T) {
	repo := newFakeAccountRepo()
	journalRepo := newMemJournalRepo()
	svc := accountingsvc.NewAccountService(repo, journalRepo)

	ob := decimal.NewFromInt(500_000)
	_, err := svc.OpenAccount(context.Background(), accountingsvc.OpenAccountCmd{
		UserID: "u1", Name: "Cash", Type: accounting.Asset, Currency: "VND",
		OpeningBalance: &ob,
	})
	if err == nil {
		t.Error("want error when Opening Balance account missing")
	}
}

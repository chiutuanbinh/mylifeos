package accountingsvc_test

import (
	"context"
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
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

func TestAccountService_OpenAccount_Root(t *testing.T) {
	repo := newFakeAccountRepo()
	svc := accountingsvc.NewAccountService(repo)

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
	svc := accountingsvc.NewAccountService(repo)

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
	svc := accountingsvc.NewAccountService(repo)

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

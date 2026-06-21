package accountingsvc

import (
	"context"
	"errors"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
)

type AccountService struct {
	accounts repository.AccountRepo
	journal  repository.JournalRepo
}

func NewAccountService(accounts repository.AccountRepo, journal repository.JournalRepo) *AccountService {
	return &AccountService{accounts: accounts, journal: journal}
}

func (s *AccountService) OpenAccount(ctx context.Context, cmd OpenAccountCmd) (accounting.AccountID, error) {
	if cmd.ParentID != nil {
		parent, err := s.accounts.FindByID(ctx, accounting.AccountID(*cmd.ParentID))
		if err != nil {
			return "", err
		}
		if parent.UserID() != cmd.UserID {
			return "", errors.New("parent account not found")
		}
		if !parent.IsGroup() {
			return "", errors.New("parent account must be a group")
		}
	}
	if cmd.Currency == "" {
		cmd.Currency = "VND"
	}
	a := accounting.NewAccount(cmd.UserID, cmd.ParentID, cmd.Name, cmd.Type, cmd.Currency, cmd.IsGroup, cmd.SortOrder)
	if cmd.AssetMeta != nil {
		a.AttachAssetMeta(&accounting.AssetMeta{
			PurchaseValue:    cmd.AssetMeta.PurchaseValue,
			PurchasedAt:      cmd.AssetMeta.PurchasedAt,
			DepreciationRate: cmd.AssetMeta.DepreciationRate,
			Notes:            cmd.AssetMeta.Notes,
		})
	}
	if err := s.accounts.Save(ctx, a); err != nil {
		return "", err
	}
	if cmd.OpeningBalance != nil && cmd.OpeningBalance.IsPositive() {
		ob, err := s.accounts.FindByNameAndType(ctx, cmd.UserID, "Opening Balance", accounting.Equity)
		if err != nil {
			return "", errors.New("Opening Balance equity account not found; run account setup first")
		}
		entry := accounting.NewJournalEntry(cmd.UserID, time.Now(), "Opening balance — "+cmd.Name)
		_ = entry.AddLine(a.ID(), accounting.Money{Amount: *cmd.OpeningBalance, Currency: cmd.Currency}, accounting.Debit)
		_ = entry.AddLine(ob.ID(), accounting.Money{Amount: *cmd.OpeningBalance, Currency: cmd.Currency}, accounting.Credit)
		if err := s.journal.Save(ctx, entry); err != nil {
			return "", err
		}
	}
	return a.ID(), nil
}

func (s *AccountService) UpdateAccount(ctx context.Context, cmd UpdateAccountCmd) error {
	a, err := s.accounts.FindByID(ctx, accounting.AccountID(cmd.ID))
	if err != nil {
		return err
	}
	if a.UserID() != cmd.UserID {
		return errors.New("account not found")
	}
	if cmd.ParentID != nil {
		parent, err := s.accounts.FindByID(ctx, accounting.AccountID(*cmd.ParentID))
		if err != nil {
			return errors.New("parent account not found")
		}
		if parent.UserID() != cmd.UserID {
			return errors.New("parent account not found")
		}
		if !parent.IsGroup() {
			return errors.New("parent account must be a group")
		}
		pid := accounting.AccountID(*cmd.ParentID)
		a.Reparent(&pid)
	} else {
		a.Reparent(nil)
	}
	a.Rename(cmd.Name)
	a.ChangeType(cmd.Type)
	a.Reorder(cmd.SortOrder)
	if cmd.AssetMeta != nil {
		a.AttachAssetMeta(&accounting.AssetMeta{
			PurchaseValue:    cmd.AssetMeta.PurchaseValue,
			PurchasedAt:      cmd.AssetMeta.PurchasedAt,
			DepreciationRate: cmd.AssetMeta.DepreciationRate,
			Notes:            cmd.AssetMeta.Notes,
		})
	} else {
		a.AttachAssetMeta(nil)
	}
	return s.accounts.Save(ctx, a)
}

func (s *AccountService) ListAccounts(ctx context.Context, userID string) ([]*accounting.Account, error) {
	return s.accounts.FindByUser(ctx, userID)
}

var (
	ErrAccountHasChildren     = errors.New("account has child accounts")
	ErrAccountHasJournalLines = errors.New("account has journal entries")
)

func (s *AccountService) DeleteAccount(ctx context.Context, userID, id string) error {
	acctID := accounting.AccountID(id)
	// verify ownership
	acct, err := s.accounts.FindByID(ctx, acctID)
	if err != nil {
		return err
	}
	if acct.UserID() != userID {
		return repository.ErrAccountNotFound
	}
	// check children
	all, err := s.accounts.FindByUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, a := range all {
		if a.ParentID() != nil && *a.ParentID() == acctID {
			return ErrAccountHasChildren
		}
	}
	// check journal lines
	entries, err := s.journal.FindByUser(ctx, userID, time.Time{}, time.Now())
	if err != nil {
		return err
	}
	for _, e := range entries {
		for _, l := range e.Lines() {
			if l.AccountID() == acctID {
				return ErrAccountHasJournalLines
			}
		}
	}
	return s.accounts.Delete(ctx, acctID)
}

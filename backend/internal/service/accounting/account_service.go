package accountingsvc

import (
	"context"
	"errors"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
)

type AccountService struct {
	accounts repository.AccountRepo
}

func NewAccountService(accounts repository.AccountRepo) *AccountService {
	return &AccountService{accounts: accounts}
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
	a := accounting.NewAccount(cmd.UserID, cmd.ParentID, cmd.Name, cmd.Type, cmd.Currency, cmd.IsGroup, cmd.SortOrder)
	if err := s.accounts.Save(ctx, a); err != nil {
		return "", err
	}
	return a.ID(), nil
}

func (s *AccountService) ListAccounts(ctx context.Context, userID string) ([]*accounting.Account, error) {
	return s.accounts.FindByUser(ctx, userID)
}

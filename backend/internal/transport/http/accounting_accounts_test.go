package httphandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
	httphandler "github.com/chiutuanbinh/mylifeos/backend/internal/transport/http"
	"github.com/shopspring/decimal"
)

func mustDecimal(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

type testAccountRepo struct {
	accounts map[accounting.AccountID]*accounting.Account
}

func newTestAccountRepo() *testAccountRepo {
	return &testAccountRepo{accounts: map[accounting.AccountID]*accounting.Account{}}
}

func (r *testAccountRepo) Save(_ context.Context, a *accounting.Account) error {
	r.accounts[a.ID()] = a
	return nil
}

func (r *testAccountRepo) FindByUser(_ context.Context, _ string) ([]*accounting.Account, error) {
	var result []*accounting.Account
	for _, a := range r.accounts {
		result = append(result, a)
	}
	return result, nil
}

func (r *testAccountRepo) FindByID(_ context.Context, id accounting.AccountID) (*accounting.Account, error) {
	a, ok := r.accounts[id]
	if !ok {
		return nil, repository.ErrAccountNotFound
	}
	return a, nil
}

func (r *testAccountRepo) FindByNameAndType(_ context.Context, userID, name string, t accounting.AccountType) (*accounting.Account, error) {
	for _, a := range r.accounts {
		if a.UserID() == userID && a.Name() == name && a.Type() == t {
			return a, nil
		}
	}
	return nil, repository.ErrAccountNotFound
}

func TestAccountsHandler_Create_Success(t *testing.T) {
	svc := accountingsvc.NewAccountService(newTestAccountRepo(), nil)
	h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

	body, _ := json.Marshal(map[string]interface{}{
		"name": "Cash", "type": "asset", "currency": "VND",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/accounts", bytes.NewReader(body))
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["id"] == "" {
		t.Error("want non-empty id in response")
	}
}

func TestAccountsHandler_Create_MissingName(t *testing.T) {
	svc := accountingsvc.NewAccountService(newTestAccountRepo(), nil)
	h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

	body, _ := json.Marshal(map[string]interface{}{"type": "asset"})
	req := httptest.NewRequest(http.MethodPost, "/api/accounts", bytes.NewReader(body))
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rr.Code)
	}
}

func TestAccountsHandler_Create_MissingType(t *testing.T) {
	svc := accountingsvc.NewAccountService(newTestAccountRepo(), nil)
	h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

	body, _ := json.Marshal(map[string]interface{}{"name": "Cash"})
	req := httptest.NewRequest(http.MethodPost, "/api/accounts", bytes.NewReader(body))
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rr.Code)
	}
}

func TestAccountsHandler_Create_BadJSON(t *testing.T) {
	svc := accountingsvc.NewAccountService(newTestAccountRepo(), nil)
	h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

	req := httptest.NewRequest(http.MethodPost, "/api/accounts", bytes.NewReader([]byte("not json")))
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rr.Code)
	}
}

func TestAccountsHandler_List_Empty(t *testing.T) {
	svc := accountingsvc.NewAccountService(newTestAccountRepo(), nil)
	h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rr.Code)
	}
}

func TestAccountsHandler_List_AfterCreate(t *testing.T) {
	repo := newTestAccountRepo()
	svc := accountingsvc.NewAccountService(repo, nil)
	h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

	// create one account
	body, _ := json.Marshal(map[string]interface{}{
		"name": "Savings", "type": "asset", "currency": "VND",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/accounts", bytes.NewReader(body))
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr := httptest.NewRecorder()
	h.Create(rr, req)

	// list
	req2 := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	req2 = req2.WithContext(withUserID(req2.Context(), "user1"))
	rr2 := httptest.NewRecorder()
	h.List(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rr2.Code)
	}
	var resp []map[string]interface{}
	json.NewDecoder(rr2.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Errorf("want 1 account, got %d", len(resp))
	}
}

func TestAccountsHandler_List_BalanceFromJournal(t *testing.T) {
	repo := newTestAccountRepo()
	svc := accountingsvc.NewAccountService(repo, nil)

	// create cash account
	body, _ := json.Marshal(map[string]any{
		"name": "Cash", "type": "asset", "currency": "VND",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/accounts", bytes.NewReader(body))
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr := httptest.NewRecorder()
	h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})
	h.Create(rr, req)

	var created map[string]string
	json.NewDecoder(rr.Body).Decode(&created)
	acctID := accounting.AccountID(created["id"])

	// seed a journal entry with a debit line on cash
	jr := &testJournalRepo{}
	entry := accounting.NewJournalEntry("user1", time.Now(), "seed")
	m, _ := accounting.NewMoney(mustDecimal("500000"), "VND")
	_ = entry.AddLine(acctID, m, accounting.Debit)
	_ = entry.Post()
	jr.saved = append(jr.saved, entry)

	h2 := httphandler.NewAccountsHandler(svc, jr)
	req2 := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	req2 = req2.WithContext(withUserID(req2.Context(), "user1"))
	rr2 := httptest.NewRecorder()
	h2.List(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
	var resp []map[string]any
	json.NewDecoder(rr2.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Fatalf("want 1 account, got %d", len(resp))
	}
	if resp[0]["balance"] != float64(500000) {
		t.Errorf("want balance 500000, got %v", resp[0]["balance"])
	}
}

func TestAccountsHandler_List_GroupAggregatesBalance(t *testing.T) {
	repo := newTestAccountRepo()
	svc := accountingsvc.NewAccountService(repo, nil)
	h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})
	withCtx := func(r *http.Request) *http.Request { return r.WithContext(withUserID(r.Context(), "user1")) }

	// create group
	body, _ := json.Marshal(map[string]any{"name": "Assets", "type": "asset", "currency": "VND", "is_group": true})
	req := httptest.NewRequest(http.MethodPost, "/api/accounts", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.Create(rr, withCtx(req))
	var grp map[string]string
	json.NewDecoder(rr.Body).Decode(&grp)
	groupID := grp["id"]

	// create leaf under group
	body2, _ := json.Marshal(map[string]any{"name": "Cash", "type": "asset", "currency": "VND", "parent_id": groupID})
	req2 := httptest.NewRequest(http.MethodPost, "/api/accounts", bytes.NewReader(body2))
	rr2 := httptest.NewRecorder()
	h.Create(rr2, withCtx(req2))
	var leaf map[string]string
	json.NewDecoder(rr2.Body).Decode(&leaf)
	leafID := accounting.AccountID(leaf["id"])

	// journal entry debiting cash
	jr := &testJournalRepo{}
	entry := accounting.NewJournalEntry("user1", time.Now(), "seed")
	m, _ := accounting.NewMoney(mustDecimal("1000000"), "VND")
	_ = entry.AddLine(leafID, m, accounting.Debit)
	_ = entry.Post()
	jr.saved = append(jr.saved, entry)

	h2 := httphandler.NewAccountsHandler(svc, jr)
	req3 := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	rr3 := httptest.NewRecorder()
	h2.List(rr3, withCtx(req3))

	var resp []map[string]any
	json.NewDecoder(rr3.Body).Decode(&resp)

	balanceFor := func(name string) float64 {
		for _, r := range resp {
			if r["name"] == name {
				b, _ := r["balance"].(float64)
				return b
			}
		}
		return -1
	}
	if balanceFor("Cash") != float64(1000000) {
		t.Errorf("Cash balance: want 1000000, got %v", balanceFor("Cash"))
	}
	if balanceFor("Assets") != float64(1000000) {
		t.Errorf("Assets balance: want 1000000, got %v", balanceFor("Assets"))
	}
}

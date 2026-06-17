package httphandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
	httphandler "github.com/chiutuanbinh/mylifeos/backend/internal/transport/http"
)

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
		return nil, nil
	}
	return a, nil
}

func TestAccountsHandler_Create_Success(t *testing.T) {
	svc := accountingsvc.NewAccountService(newTestAccountRepo())
	h := httphandler.NewAccountsHandler(svc)

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
	svc := accountingsvc.NewAccountService(newTestAccountRepo())
	h := httphandler.NewAccountsHandler(svc)

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
	svc := accountingsvc.NewAccountService(newTestAccountRepo())
	h := httphandler.NewAccountsHandler(svc)

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
	svc := accountingsvc.NewAccountService(newTestAccountRepo())
	h := httphandler.NewAccountsHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/accounts", bytes.NewReader([]byte("not json")))
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rr.Code)
	}
}

func TestAccountsHandler_List_Empty(t *testing.T) {
	svc := accountingsvc.NewAccountService(newTestAccountRepo())
	h := httphandler.NewAccountsHandler(svc)

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
	svc := accountingsvc.NewAccountService(repo)
	h := httphandler.NewAccountsHandler(svc)

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

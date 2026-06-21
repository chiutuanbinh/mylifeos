package httphandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
	httphandler "github.com/chiutuanbinh/mylifeos/backend/internal/transport/http"
	"github.com/shopspring/decimal"
)

func setUserID(ctx context.Context, userID string) context.Context {
	return withUserID(ctx, userID)
}

func setChiURLParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

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

func (r *testAccountRepo) FindByID(_ context.Context, id accounting.AccountID, userID string) (*accounting.Account, error) {
	a, ok := r.accounts[id]
	if !ok {
		return nil, repository.ErrAccountNotFound
	}
	if a.UserID() != userID {
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

func (r *testAccountRepo) Delete(_ context.Context, id accounting.AccountID, userID string) error {
	delete(r.accounts, id)
	return nil
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

func TestAccountsHandler_Update_Success(t *testing.T) {
	repo := newTestAccountRepo()
	svc := accountingsvc.NewAccountService(repo, &testJournalRepo{})
	h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

	// create account
	createBody, _ := json.Marshal(map[string]interface{}{
		"name": "Old", "type": "asset", "currency": "VND", "is_group": false, "sort_order": 0,
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewReader(createBody))
	r = r.WithContext(setUserID(r.Context(), "user1"))
	h.Create(w, r)
	var created map[string]string
	json.NewDecoder(w.Body).Decode(&created)
	id := created["id"]

	// patch it
	patchBody, _ := json.Marshal(map[string]interface{}{
		"name": "New", "type": "expense", "sort_order": 2,
	})
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodPatch, "/accounts/"+id, bytes.NewReader(patchBody))
	r2 = r2.WithContext(setUserID(r2.Context(), "user1"))
	r2 = setChiURLParam(r2, "id", id)
	h.Update(w2, r2)
	if w2.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestAccountsHandler_Update_WrongUser(t *testing.T) {
	repo := newTestAccountRepo()
	svc := accountingsvc.NewAccountService(repo, &testJournalRepo{})
	h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

	createBody, _ := json.Marshal(map[string]interface{}{
		"name": "X", "type": "asset", "currency": "VND", "is_group": false, "sort_order": 0,
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewReader(createBody))
	r = r.WithContext(setUserID(r.Context(), "user1"))
	h.Create(w, r)
	var created map[string]string
	json.NewDecoder(w.Body).Decode(&created)
	id := created["id"]

	patchBody, _ := json.Marshal(map[string]interface{}{"name": "Hacked", "type": "expense"})
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodPatch, "/accounts/"+id, bytes.NewReader(patchBody))
	r2 = r2.WithContext(setUserID(r2.Context(), "user2"))
	r2 = setChiURLParam(r2, "id", id)
	h.Update(w2, r2)
	if w2.Code == http.StatusNoContent {
		t.Error("want non-204 for wrong user")
	}
}

func TestAccountsHandler_Create_WithAssetMeta(t *testing.T) {
	repo := newTestAccountRepo()
	svc := accountingsvc.NewAccountService(repo, &testJournalRepo{})
	h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

	purchasedAt := "2024-01-15"
	body, _ := json.Marshal(map[string]interface{}{
		"name": "Car", "type": "asset", "currency": "VND",
		"asset_meta": map[string]interface{}{
			"purchase_value":    500000000,
			"purchased_at":      purchasedAt,
			"depreciation_rate": 0.2,
			"notes":             "Toyota",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewReader(body))
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr := httptest.NewRecorder()
	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAccountsHandler_Update_BadJSON(t *testing.T) {
	repo := newTestAccountRepo()
	svc := accountingsvc.NewAccountService(repo, &testJournalRepo{})
	h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

	r := httptest.NewRequest(http.MethodPatch, "/accounts/foo", bytes.NewReader([]byte("bad json")))
	r = r.WithContext(setUserID(r.Context(), "user1"))
	r = setChiURLParam(r, "id", "foo")
	w := httptest.NewRecorder()
	h.Update(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestAccountsHandler_Update_MissingName(t *testing.T) {
	repo := newTestAccountRepo()
	svc := accountingsvc.NewAccountService(repo, &testJournalRepo{})
	h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

	body, _ := json.Marshal(map[string]interface{}{"type": "asset"})
	r := httptest.NewRequest(http.MethodPatch, "/accounts/foo", bytes.NewReader(body))
	r = r.WithContext(setUserID(r.Context(), "user1"))
	r = setChiURLParam(r, "id", "foo")
	w := httptest.NewRecorder()
	h.Update(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestAccountsHandler_Delete_Success(t *testing.T) {
	repo := newTestAccountRepo()
	acct := accounting.NewAccount("user1", nil, "Cash", accounting.Asset, "VND", false, 0)
	repo.accounts[acct.ID()] = acct

	svc := accountingsvc.NewAccountService(repo, &testJournalRepo{})
	h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

	r := httptest.NewRequest(http.MethodDelete, "/accounts/"+string(acct.ID()), nil)
	r = r.WithContext(setUserID(r.Context(), "user1"))
	r = setChiURLParam(r, "id", string(acct.ID()))
	w := httptest.NewRecorder()
	h.Delete(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAccountsHandler_Delete_HasChildren(t *testing.T) {
	repo := newTestAccountRepo()
	parent := accounting.NewAccount("user1", nil, "Assets", accounting.Asset, "VND", true, 0)
	parentIDStr := string(parent.ID())
	child := accounting.NewAccount("user1", &parentIDStr, "Cash", accounting.Asset, "VND", false, 0)
	repo.accounts[parent.ID()] = parent
	repo.accounts[child.ID()] = child

	svc := accountingsvc.NewAccountService(repo, &testJournalRepo{})
	h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

	r := httptest.NewRequest(http.MethodDelete, "/accounts/"+string(parent.ID()), nil)
	r = r.WithContext(setUserID(r.Context(), "user1"))
	r = setChiURLParam(r, "id", string(parent.ID()))
	w := httptest.NewRecorder()
	h.Delete(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAccountsHandler_Delete_HasJournalLines(t *testing.T) {
	repo := newTestAccountRepo()
	acct := accounting.NewAccount("user1", nil, "Cash", accounting.Asset, "VND", false, 0)
	repo.accounts[acct.ID()] = acct

	jr := &testJournalRepo{}
	entry := accounting.NewJournalEntry("user1", time.Now(), "test")
	_ = entry.AddLine(acct.ID(), accounting.Money{Amount: decimal.NewFromInt(100), Currency: "VND"}, accounting.Debit)
	_ = entry.Post()
	jr.saved = append(jr.saved, entry)

	svc := accountingsvc.NewAccountService(repo, jr)
	h := httphandler.NewAccountsHandler(svc, jr)

	r := httptest.NewRequest(http.MethodDelete, "/accounts/"+string(acct.ID()), nil)
	r = r.WithContext(setUserID(r.Context(), "user1"))
	r = setChiURLParam(r, "id", string(acct.ID()))
	w := httptest.NewRecorder()
	h.Delete(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAccountsHandler_Delete_NotFound(t *testing.T) {
	repo := newTestAccountRepo()
	svc := accountingsvc.NewAccountService(repo, &testJournalRepo{})
	h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

	r := httptest.NewRequest(http.MethodDelete, "/accounts/nonexistent", nil)
	r = r.WithContext(setUserID(r.Context(), "user1"))
	r = setChiURLParam(r, "id", "nonexistent")
	w := httptest.NewRecorder()
	h.Delete(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAccountsHandler_Update_WithAssetMeta(t *testing.T) {
	repo := newTestAccountRepo()
	svc := accountingsvc.NewAccountService(repo, &testJournalRepo{})
	h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

	// create account
	createBody, _ := json.Marshal(map[string]interface{}{
		"name": "House", "type": "asset", "currency": "VND",
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewReader(createBody))
	r = r.WithContext(withUserID(r.Context(), "user1"))
	h.Create(w, r)
	var created map[string]string
	json.NewDecoder(w.Body).Decode(&created)
	id := created["id"]

	patchBody, _ := json.Marshal(map[string]interface{}{
		"name": "House Updated", "type": "asset",
		"asset_meta": map[string]interface{}{
			"purchase_value":    1000000000,
			"purchased_at":      "2023-06-01",
			"depreciation_rate": 0.05,
			"notes":             "main house",
		},
	})
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodPatch, "/accounts/"+id, bytes.NewReader(patchBody))
	r2 = r2.WithContext(setUserID(r2.Context(), "user1"))
	r2 = setChiURLParam(r2, "id", id)
	h.Update(w2, r2)
	if w2.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d: %s", w2.Code, w2.Body.String())
	}
}

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
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
	httphandler "github.com/chiutuanbinh/mylifeos/backend/internal/transport/http"
	"github.com/shopspring/decimal"
)

type testJournalRepo struct {
	saved     []*accounting.JournalEntry
	goalLinks map[string][]string // entryID -> goalIDs
}

func (r *testJournalRepo) Save(_ context.Context, e *accounting.JournalEntry) error {
	r.saved = append(r.saved, e)
	return nil
}

func (r *testJournalRepo) FindByUser(_ context.Context, _ string, _, _ time.Time) ([]*accounting.JournalEntry, error) {
	return r.saved, nil
}

func (r *testJournalRepo) SaveGoalLinks(_ context.Context, entryID, _ string, goalIDs []string) error {
	if r.goalLinks == nil {
		r.goalLinks = map[string][]string{}
	}
	r.goalLinks[entryID] = goalIDs
	return nil
}

type testPublisher struct{}

func (p *testPublisher) Publish(_ context.Context, _ accounting.DomainEvent) error { return nil }

// newTestAccountRepoWithIDs returns a testAccountRepo pre-populated with accounts having the given IDs.
func newTestAccountRepoWithIDs(userID string, ids ...string) *testAccountRepo {
	r := newTestAccountRepo()
	for _, id := range ids {
		a := accounting.ReconstituteAccount(id, userID, nil, id, accounting.Asset, "VND", false, false, 0)
		r.accounts[a.ID()] = a
	}
	return r
}

func TestJournalHandler_RecordTransaction_WithGoalIDs(t *testing.T) {
	jRepo := &testJournalRepo{}
	aRepo := newTestAccountRepoWithIDs("user1", "acc-food", "acc-visa")
	pub := &testPublisher{}

	journalSvc := accountingsvc.NewJournalService(jRepo, aRepo, pub)
	nwQuery := accountingsvc.NewNetWorthQuery(aRepo, jRepo)
	h := httphandler.NewJournalHandler(journalSvc, nwQuery)

	body, _ := json.Marshal(map[string]interface{}{
		"date":        "2026-07-01",
		"description": "Coffee",
		"goal_ids":    []string{"goal-1", "goal-2"},
		"lines": []map[string]interface{}{
			{"account_id": "acc-food", "amount": 150000, "side": "debit"},
			{"account_id": "acc-visa", "amount": 150000, "side": "credit"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/journal/entries", bytes.NewReader(body))
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr := httptest.NewRecorder()
	h.RecordTransaction(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if len(jRepo.goalLinks) == 0 {
		t.Error("expected goal links to be saved")
	}
}

func TestJournalHandler_RecordTransaction_Balanced(t *testing.T) {
	jRepo := &testJournalRepo{}
	aRepo := newTestAccountRepoWithIDs("user1", "acc-food", "acc-visa")
	pub := &testPublisher{}

	journalSvc := accountingsvc.NewJournalService(jRepo, aRepo, pub)
	nwQuery := accountingsvc.NewNetWorthQuery(aRepo, jRepo)
	h := httphandler.NewJournalHandler(journalSvc, nwQuery)

	body, _ := json.Marshal(map[string]interface{}{
		"date":        "2026-07-01",
		"description": "Coffee",
		"lines": []map[string]interface{}{
			{"account_id": "acc-food", "amount": 150000, "side": "debit"},
			{"account_id": "acc-visa", "amount": 150000, "side": "credit"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/journal/entries", bytes.NewReader(body))
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr := httptest.NewRecorder()

	h.RecordTransaction(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestJournalHandler_RecordTransaction_UnbalancedReturns422(t *testing.T) {
	jRepo := &testJournalRepo{}
	aRepo := newTestAccountRepoWithIDs("user1", "a", "b")
	pub := &testPublisher{}

	journalSvc := accountingsvc.NewJournalService(jRepo, aRepo, pub)
	nwQuery := accountingsvc.NewNetWorthQuery(aRepo, jRepo)
	h := httphandler.NewJournalHandler(journalSvc, nwQuery)

	body, _ := json.Marshal(map[string]interface{}{
		"date":        "2026-07-01",
		"description": "Bad",
		"lines": []map[string]interface{}{
			{"account_id": "a", "amount": 100, "side": "debit"},
			{"account_id": "b", "amount": 50, "side": "credit"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/journal/entries", bytes.NewReader(body))
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr := httptest.NewRecorder()

	h.RecordTransaction(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d", rr.Code)
	}
}

func TestJournalHandler_RecordTransaction_BadJSON(t *testing.T) {
	jRepo := &testJournalRepo{}
	aRepo := newTestAccountRepo()
	pub := &testPublisher{}

	journalSvc := accountingsvc.NewJournalService(jRepo, aRepo, pub)
	nwQuery := accountingsvc.NewNetWorthQuery(aRepo, jRepo)
	h := httphandler.NewJournalHandler(journalSvc, nwQuery)

	req := httptest.NewRequest(http.MethodPost, "/api/journal/entries", bytes.NewReader([]byte("not json")))
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr := httptest.NewRecorder()

	h.RecordTransaction(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rr.Code)
	}
}

func TestJournalHandler_RecordTransaction_BadDate(t *testing.T) {
	jRepo := &testJournalRepo{}
	aRepo := newTestAccountRepo()
	pub := &testPublisher{}

	journalSvc := accountingsvc.NewJournalService(jRepo, aRepo, pub)
	nwQuery := accountingsvc.NewNetWorthQuery(aRepo, jRepo)
	h := httphandler.NewJournalHandler(journalSvc, nwQuery)

	body, _ := json.Marshal(map[string]interface{}{
		"date": "not-a-date",
		"lines": []map[string]interface{}{
			{"account_id": "a", "amount": 100, "side": "debit"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/journal/entries", bytes.NewReader(body))
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr := httptest.NewRecorder()

	h.RecordTransaction(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rr.Code)
	}
}

func TestJournalHandler_ListEntries_ReturnsJSON(t *testing.T) {
	jRepo := &testJournalRepo{}
	aRepo := newTestAccountRepoWithIDs("user1", "acc-food", "acc-visa")
	pub := &testPublisher{}

	journalSvc := accountingsvc.NewJournalService(jRepo, aRepo, pub)
	nwQuery := accountingsvc.NewNetWorthQuery(aRepo, jRepo)
	h := httphandler.NewJournalHandler(journalSvc, nwQuery)

	// seed one entry via RecordTransaction so the repo has data
	seedBody, _ := json.Marshal(map[string]interface{}{
		"date":        "2026-07-01",
		"description": "Coffee",
		"lines": []map[string]interface{}{
			{"account_id": "acc-food", "amount": 50000, "side": "debit"},
			{"account_id": "acc-visa", "amount": 50000, "side": "credit"},
		},
	})
	seedReq := httptest.NewRequest(http.MethodPost, "/api/journal/entries", bytes.NewReader(seedBody))
	seedReq = seedReq.WithContext(withUserID(seedReq.Context(), "user1"))
	h.RecordTransaction(httptest.NewRecorder(), seedReq)

	req := httptest.NewRequest(http.MethodGet, "/api/journal/entries", nil)
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr := httptest.NewRecorder()

	h.ListEntries(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var entries []map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&entries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("want 1 entry, got %d", len(entries))
	}
	if entries[0]["description"] != "Coffee" {
		t.Errorf("want description Coffee, got %v", entries[0]["description"])
	}
	lines, ok := entries[0]["lines"].([]interface{})
	if !ok || len(lines) != 2 {
		t.Errorf("want 2 lines, got %v", entries[0]["lines"])
	}
}

func TestJournalHandler_ListEntries_IncludesGoalIDs(t *testing.T) {
	entry := accounting.ReconstituteEntry("e1", "user1", time.Now(), "desc", "")
	entry.SetGoalIDs([]string{"g1", "g2"})
	entry.ReconstituteLine("l1", "acc1", accounting.Money{Amount: decimal.NewFromInt(1), Currency: "VND"}, accounting.Debit)
	entry.ReconstituteLine("l2", "acc2", accounting.Money{Amount: decimal.NewFromInt(1), Currency: "VND"}, accounting.Credit)

	jRepo := &testJournalRepo{saved: []*accounting.JournalEntry{entry}}
	aRepo := newTestAccountRepoWithIDs("user1", "acc1", "acc2")
	pub := &testPublisher{}

	journalSvc := accountingsvc.NewJournalService(jRepo, aRepo, pub)
	nwQuery := accountingsvc.NewNetWorthQuery(aRepo, jRepo)
	h := httphandler.NewJournalHandler(journalSvc, nwQuery)

	req := httptest.NewRequest(http.MethodGet, "/api/journal/entries", nil)
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr := httptest.NewRecorder()
	h.ListEntries(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var result []map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)
	if len(result) == 0 {
		t.Fatal("expected entries")
	}
	goalIDs, ok := result[0]["goal_ids"].([]interface{})
	if !ok || len(goalIDs) != 2 {
		t.Errorf("expected 2 goal_ids, got %v", result[0]["goal_ids"])
	}
}

func TestJournalHandler_NetWorth_ReturnsJSON(t *testing.T) {
	jRepo := &testJournalRepo{}
	aRepo := newTestAccountRepo()
	pub := &testPublisher{}

	journalSvc := accountingsvc.NewJournalService(jRepo, aRepo, pub)
	nwQuery := accountingsvc.NewNetWorthQuery(aRepo, jRepo)
	h := httphandler.NewJournalHandler(journalSvc, nwQuery)

	req := httptest.NewRequest(http.MethodGet, "/api/journal/networth", nil)
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr := httptest.NewRecorder()

	h.NetWorth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rr.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if _, ok := resp["net_worth"]; !ok {
		t.Error("want net_worth in response")
	}
}

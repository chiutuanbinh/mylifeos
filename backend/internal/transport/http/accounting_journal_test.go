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
)

type testJournalRepo struct {
	saved []*accounting.JournalEntry
}

func (r *testJournalRepo) Save(_ context.Context, e *accounting.JournalEntry) error {
	r.saved = append(r.saved, e)
	return nil
}

func (r *testJournalRepo) FindByUser(_ context.Context, _ string, _, _ time.Time) ([]*accounting.JournalEntry, error) {
	return r.saved, nil
}

type testPublisher struct{}

func (p *testPublisher) Publish(_ context.Context, _ accounting.DomainEvent) error { return nil }

func TestJournalHandler_RecordTransaction_Balanced(t *testing.T) {
	jRepo := &testJournalRepo{}
	aRepo := newTestAccountRepo()
	pub := &testPublisher{}

	journalSvc := accountingsvc.NewJournalService(jRepo, pub)
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
	aRepo := newTestAccountRepo()
	pub := &testPublisher{}

	journalSvc := accountingsvc.NewJournalService(jRepo, pub)
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

	journalSvc := accountingsvc.NewJournalService(jRepo, pub)
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

	journalSvc := accountingsvc.NewJournalService(jRepo, pub)
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

func TestJournalHandler_NetWorth_ReturnsJSON(t *testing.T) {
	jRepo := &testJournalRepo{}
	aRepo := newTestAccountRepo()
	pub := &testPublisher{}

	journalSvc := accountingsvc.NewJournalService(jRepo, pub)
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

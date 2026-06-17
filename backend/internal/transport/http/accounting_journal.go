package httphandler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
	"github.com/shopspring/decimal"
)

type JournalHandler struct {
	journal  *accountingsvc.JournalService
	networth *accountingsvc.NetWorthQuery
}

func NewJournalHandler(journal *accountingsvc.JournalService, networth *accountingsvc.NetWorthQuery) *JournalHandler {
	return &JournalHandler{journal: journal, networth: networth}
}

func (h *JournalHandler) RecordTransaction(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	var req struct {
		Date        string `json:"date"`
		Description string `json:"description"`
		Memo        string `json:"memo"`
		Lines       []struct {
			AccountID string          `json:"account_id"`
			Amount    decimal.Decimal `json:"amount"`
			Currency  string          `json:"currency"`
			Side      string          `json:"side"`
		} `json:"lines"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		http.Error(w, "invalid date format, use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	lines := make([]accountingsvc.LineCmd, len(req.Lines))
	for i, l := range req.Lines {
		cur := l.Currency
		if cur == "" {
			cur = "VND"
		}
		lines[i] = accountingsvc.LineCmd{
			AccountID: l.AccountID,
			Amount:    l.Amount,
			Currency:  cur,
			Side:      accounting.Side(l.Side),
		}
	}
	cmd := accountingsvc.RecordTransactionCmd{
		UserID:      userID,
		Date:        date,
		Description: req.Description,
		Memo:        req.Memo,
		Lines:       lines,
	}
	id, err := h.journal.RecordTransaction(r.Context(), cmd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": string(id)})
}

func (h *JournalHandler) NetWorth(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	nw, err := h.networth.Current(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"net_worth": nw.Amount,
		"currency":  nw.Currency,
	})
}

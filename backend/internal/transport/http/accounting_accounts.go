package httphandler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
	"github.com/shopspring/decimal"
)

type AccountsHandler struct {
	svc     *accountingsvc.AccountService
	journal repository.JournalRepo
}

func NewAccountsHandler(svc *accountingsvc.AccountService, journal repository.JournalRepo) *AccountsHandler {
	return &AccountsHandler{svc: svc, journal: journal}
}

func (h *AccountsHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	accounts, err := h.svc.ListAccounts(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	entries, err := h.journal.FindByUser(r.Context(), userID, time.Time{}, time.Now())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	var allLines []accounting.JournalLine
	for _, e := range entries {
		allLines = append(allLines, e.Lines()...)
	}

	// compute leaf balances
	leafBalance := map[string]string{}
	for _, a := range accounts {
		if !a.IsGroup() {
			m, err := a.Balance(allLines)
			if err == nil {
				leafBalance[string(a.ID())] = m.Amount.String()
			}
		}
	}

	// build parent→children index for group aggregation
	children := map[string][]string{}
	for _, a := range accounts {
		if a.ParentID() != nil {
			pid := string(*a.ParentID())
			children[pid] = append(children[pid], string(a.ID()))
		}
	}

	// sum leaf descendants for a group (recursive)
	var sumDescendants func(id string) float64
	sumDescendants = func(id string) float64 {
		if bal, ok := leafBalance[id]; ok {
			v, _ := decimal.NewFromString(bal)
			f, _ := v.Float64()
			return f
		}
		var total float64
		for _, cid := range children[id] {
			total += sumDescendants(cid)
		}
		return total
	}

	type row struct {
		ID        string  `json:"id"`
		ParentID  string  `json:"parent_id,omitempty"`
		Name      string  `json:"name"`
		Type      string  `json:"type"`
		Currency  string  `json:"currency"`
		IsGroup   bool    `json:"is_group"`
		SortOrder int     `json:"sort_order"`
		Balance   float64 `json:"balance"`
	}
	resp := make([]row, len(accounts))
	for i, a := range accounts {
		var pid string
		if a.ParentID() != nil {
			pid = string(*a.ParentID())
		}
		var bal float64
		if a.IsGroup() {
			bal = sumDescendants(string(a.ID()))
		} else {
			if s, ok := leafBalance[string(a.ID())]; ok {
				d, _ := decimal.NewFromString(s)
				bal, _ = d.Float64()
			}
		}
		resp[i] = row{
			ID:        string(a.ID()),
			ParentID:  pid,
			Name:      a.Name(),
			Type:      string(a.Type()),
			Currency:  a.Currency(),
			IsGroup:   a.IsGroup(),
			SortOrder: a.SortOrder(),
			Balance:   bal,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AccountsHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	var req struct {
		ParentID  *string `json:"parent_id"`
		Name      string  `json:"name"`
		Type      string  `json:"type"`
		Currency  string  `json:"currency"`
		IsGroup   bool    `json:"is_group"`
		SortOrder int     `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Type == "" {
		http.Error(w, "name and type required", http.StatusBadRequest)
		return
	}
	if req.Currency == "" {
		req.Currency = "VND"
	}
	cmd := accountingsvc.OpenAccountCmd{
		UserID:    userID,
		ParentID:  req.ParentID,
		Name:      req.Name,
		Type:      accounting.AccountType(req.Type),
		Currency:  req.Currency,
		IsGroup:   req.IsGroup,
		SortOrder: req.SortOrder,
	}
	id, err := h.svc.OpenAccount(r.Context(), cmd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": string(id)})
}

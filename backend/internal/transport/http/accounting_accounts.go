package httphandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
)

type AccountsHandler struct {
	svc     *accountingsvc.AccountService
	journal repository.JournalRepo
}

func NewAccountsHandler(svc *accountingsvc.AccountService, journal repository.JournalRepo) *AccountsHandler {
	return &AccountsHandler{svc: svc, journal: journal}
}

type assetMetaResponse struct {
	PurchaseValue    *string `json:"purchase_value,omitempty"`
	PurchasedAt      *string `json:"purchased_at,omitempty"`
	DepreciationRate *string `json:"depreciation_rate,omitempty"`
	Notes            string  `json:"notes,omitempty"`
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

	leafBalance := map[string]string{}
	for _, a := range accounts {
		if !a.IsGroup() {
			m, err := a.Balance(allLines)
			if err == nil {
				leafBalance[string(a.ID())] = m.Amount.String()
			}
		}
	}

	children := map[string][]string{}
	for _, a := range accounts {
		if a.ParentID() != nil {
			pid := string(*a.ParentID())
			children[pid] = append(children[pid], string(a.ID()))
		}
	}

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
		ID        string             `json:"id"`
		ParentID  string             `json:"parent_id,omitempty"`
		Name      string             `json:"name"`
		Type      string             `json:"type"`
		Currency  string             `json:"currency"`
		IsGroup   bool               `json:"is_group"`
		SortOrder int                `json:"sort_order"`
		Balance   float64            `json:"balance"`
		AssetMeta *assetMetaResponse `json:"asset_meta,omitempty"`
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
		var amr *assetMetaResponse
		if m := a.AssetMeta(); m != nil {
			amr = &assetMetaResponse{Notes: m.Notes}
			if m.PurchaseValue != nil {
				s := m.PurchaseValue.String()
				amr.PurchaseValue = &s
			}
			if m.PurchasedAt != nil {
				s := m.PurchasedAt.Format("2006-01-02")
				amr.PurchasedAt = &s
			}
			if m.DepreciationRate != nil {
				s := m.DepreciationRate.String()
				amr.DepreciationRate = &s
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
			AssetMeta: amr,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AccountsHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	var req struct {
		ParentID       *string  `json:"parent_id"`
		Name           string   `json:"name"`
		Type           string   `json:"type"`
		Currency       string   `json:"currency"`
		IsGroup        bool     `json:"is_group"`
		SortOrder      int      `json:"sort_order"`
		OpeningBalance *float64 `json:"opening_balance"`
		AssetMeta      *struct {
			PurchaseValue    *float64 `json:"purchase_value"`
			PurchasedAt      *string  `json:"purchased_at"`
			DepreciationRate *float64 `json:"depreciation_rate"`
			Notes            string   `json:"notes"`
		} `json:"asset_meta"`
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
	if req.OpeningBalance != nil && *req.OpeningBalance > 0 {
		ob := decimal.NewFromFloat(*req.OpeningBalance)
		cmd.OpeningBalance = &ob
	}
	if req.AssetMeta != nil {
		amc := &accountingsvc.AssetMetaCmd{Notes: req.AssetMeta.Notes}
		if req.AssetMeta.PurchaseValue != nil {
			pv := decimal.NewFromFloat(*req.AssetMeta.PurchaseValue)
			amc.PurchaseValue = &pv
		}
		if req.AssetMeta.PurchasedAt != nil {
			t, err := time.Parse("2006-01-02", *req.AssetMeta.PurchasedAt)
			if err == nil {
				amc.PurchasedAt = &t
			}
		}
		if req.AssetMeta.DepreciationRate != nil {
			dr := decimal.NewFromFloat(*req.AssetMeta.DepreciationRate)
			amc.DepreciationRate = &dr
		}
		cmd.AssetMeta = amc
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

func (h *AccountsHandler) Update(w http.ResponseWriter, r *http.Request) {
	// PATCH uses full-replace semantics: all fields including parent_id must be
	// sent on every request. Omitting parent_id detaches the account from its parent.
	userID := middleware.GetUserID(r)
	id := chi.URLParam(r, "id")
	var req struct {
		Name      string   `json:"name"`
		Type      string   `json:"type"`
		ParentID  *string  `json:"parent_id"`
		SortOrder int      `json:"sort_order"`
		AssetMeta *struct {
			PurchaseValue    *float64 `json:"purchase_value"`
			PurchasedAt      *string  `json:"purchased_at"`
			DepreciationRate *float64 `json:"depreciation_rate"`
			Notes            string   `json:"notes"`
		} `json:"asset_meta"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Type == "" {
		http.Error(w, "name and type required", http.StatusBadRequest)
		return
	}

	cmd := accountingsvc.UpdateAccountCmd{
		ID:        id,
		UserID:    userID,
		Name:      req.Name,
		Type:      accounting.AccountType(req.Type),
		ParentID:  req.ParentID,
		SortOrder: req.SortOrder,
	}
	if req.AssetMeta != nil {
		amc := &accountingsvc.AssetMetaCmd{Notes: req.AssetMeta.Notes}
		if req.AssetMeta.PurchaseValue != nil {
			pv := decimal.NewFromFloat(*req.AssetMeta.PurchaseValue)
			amc.PurchaseValue = &pv
		}
		if req.AssetMeta.PurchasedAt != nil {
			t, err := time.Parse("2006-01-02", *req.AssetMeta.PurchasedAt)
			if err == nil {
				amc.PurchasedAt = &t
			}
		}
		if req.AssetMeta.DepreciationRate != nil {
			dr := decimal.NewFromFloat(*req.AssetMeta.DepreciationRate)
			amc.DepreciationRate = &dr
		}
		cmd.AssetMeta = amc
	}

	if err := h.svc.UpdateAccount(r.Context(), cmd); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AccountsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	id := chi.URLParam(r, "id")
	err := h.svc.DeleteAccount(r.Context(), userID, id)
	if err == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	switch {
	case errors.Is(err, repository.ErrAccountNotFound):
		http.Error(w, `{"error":"account not found"}`, http.StatusNotFound)
	case errors.Is(err, accountingsvc.ErrAccountHasChildren):
		http.Error(w, `{"error":"account has child accounts"}`, http.StatusBadRequest)
	case errors.Is(err, accountingsvc.ErrAccountHasJournalLines):
		http.Error(w, `{"error":"account has journal entries"}`, http.StatusBadRequest)
	default:
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
	}
}

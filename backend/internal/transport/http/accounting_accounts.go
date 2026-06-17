package httphandler

import (
	"encoding/json"
	"net/http"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
)

type AccountsHandler struct {
	svc *accountingsvc.AccountService
}

func NewAccountsHandler(svc *accountingsvc.AccountService) *AccountsHandler {
	return &AccountsHandler{svc: svc}
}

func (h *AccountsHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	accounts, err := h.svc.ListAccounts(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	type row struct {
		ID        string `json:"id"`
		ParentID  string `json:"parent_id,omitempty"`
		Name      string `json:"name"`
		Type      string `json:"type"`
		Currency  string `json:"currency"`
		IsGroup   bool   `json:"is_group"`
		SortOrder int    `json:"sort_order"`
	}
	resp := make([]row, len(accounts))
	for i, a := range accounts {
		var pid string
		if a.ParentID() != nil {
			pid = string(*a.ParentID())
		}
		resp[i] = row{
			ID:        string(a.ID()),
			ParentID:  pid,
			Name:      a.Name(),
			Type:      string(a.Type()),
			Currency:  a.Currency(),
			IsGroup:   a.IsGroup(),
			SortOrder: a.SortOrder(),
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

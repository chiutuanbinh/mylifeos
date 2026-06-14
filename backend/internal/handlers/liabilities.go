package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/chiutuanbinh/mylifeos/backend/internal/repo"
)

type LiabilityHandler struct{ repo repo.LiabilityRepo }

func NewLiabilityHandler(r repo.LiabilityRepo) *LiabilityHandler { return &LiabilityHandler{r} }

func validateLiability(l models.Liability) string {
	if l.Name == "" {
		return "name is required"
	}
	if l.Category == "" {
		return "category is required"
	}
	if l.Balance < 0 {
		return "balance must be >= 0"
	}
	if l.InterestRate != nil && (*l.InterestRate < 0 || *l.InterestRate > 1) {
		return "interest_rate must be between 0 and 1"
	}
	return ""
}

func (h *LiabilityHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	items, err := h.repo.List(r.Context(), uid)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *LiabilityHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var l models.Liability
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	if msg := validateLiability(l); msg != "" {
		http.Error(w, `{"error":"`+msg+`"}`, 400)
		return
	}
	l.UserID = uid
	out, err := h.repo.Create(r.Context(), l)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(out)
}

func (h *LiabilityHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var l models.Liability
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	if msg := validateLiability(l); msg != "" {
		http.Error(w, `{"error":"`+msg+`"}`, 400)
		return
	}
	l.ID = chi.URLParam(r, "id")
	l.UserID = uid
	out, err := h.repo.Update(r.Context(), l)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *LiabilityHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	if err := h.repo.Delete(r.Context(), chi.URLParam(r, "id"), uid); err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.WriteHeader(204)
}

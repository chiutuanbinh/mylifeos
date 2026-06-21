package httphandler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/finance"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
)

type TransactionHandler struct{ repo repository.TransactionRepo }

func NewTransactionHandler(r repository.TransactionRepo) *TransactionHandler {
	return &TransactionHandler{r}
}

func (h *TransactionHandler) ListBudgets(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	bs, err := h.repo.ListBudgets(r.Context(), uid)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bs)
}

func (h *TransactionHandler) DeleteBudget(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	category := chi.URLParam(r, "category")
	err := h.repo.DeleteBudget(r.Context(), uid, category)
	if errors.Is(err, repository.ErrBudgetNotFound) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TransactionHandler) UpsertBudget(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	category := chi.URLParam(r, "category")
	var b finance.Budget
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}
	b.UserID = uid
	b.Category = category
	out, err := h.repo.UpsertBudget(r.Context(), b)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

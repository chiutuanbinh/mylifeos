package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/chiutuanbinh/mylifeos/backend/internal/repo"
)

type TransactionHandler struct{ repo repo.TransactionRepo }

func NewTransactionHandler(r repo.TransactionRepo) *TransactionHandler { return &TransactionHandler{r} }

func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	txs, err := h.repo.List(r.Context(), uid, q.Get("category"), q.Get("from"), q.Get("to"), limit, offset)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txs)
}

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var t models.Transaction
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}
	t.UserID = uid
	out, err := h.repo.Create(r.Context(), t)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(out)
}

func (h *TransactionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	id := chi.URLParam(r, "id")
	if err := h.repo.Delete(r.Context(), id, uid); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

func (h *TransactionHandler) UpsertBudget(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	category := chi.URLParam(r, "category")
	var b models.Budget
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

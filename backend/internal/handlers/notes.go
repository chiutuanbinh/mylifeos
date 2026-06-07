package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/chiutuanbinh/mylifeos/backend/internal/repo"
)

type NoteHandler struct{ repo repo.NoteRepo }

func NewNoteHandler(r repo.NoteRepo) *NoteHandler { return &NoteHandler{r} }

func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	q := r.URL.Query()
	var pinned *bool
	if p := q.Get("pinned"); p == "true" {
		t := true
		pinned = &t
	} else if p == "false" {
		f := false
		pinned = &f
	}
	notes, err := h.repo.List(r.Context(), uid, q.Get("search"), q.Get("tags"), pinned)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notes)
}

func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var n models.Note
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	n.UserID = uid
	out, err := h.repo.Create(r.Context(), n)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(out)
}

func (h *NoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var n models.Note
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	n.ID = chi.URLParam(r, "id")
	n.UserID = uid
	out, err := h.repo.Update(r.Context(), n)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	if err := h.repo.Delete(r.Context(), chi.URLParam(r, "id"), uid); err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.WriteHeader(204)
}

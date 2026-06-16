package httphandler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	notesdomain "github.com/chiutuanbinh/mylifeos/backend/internal/domain/notes"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
)

type NoteHandler struct{ repo repository.NoteRepo }

func NewNoteHandler(r repository.NoteRepo) *NoteHandler { return &NoteHandler{r} }

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
	notesList, err := h.repo.List(r.Context(), uid, q.Get("search"), q.Get("tags"), pinned)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notesList)
}

func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var n notesdomain.Note
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
	id := chi.URLParam(r, "id")

	existing, err := h.repo.Get(r.Context(), id, uid)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, 404)
		return
	}

	var patch map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}

	merged := existing
	if v, ok := patch["title"]; ok {
		json.Unmarshal(v, &merged.Title)
	}
	if v, ok := patch["content"]; ok {
		json.Unmarshal(v, &merged.Content)
	}
	if v, ok := patch["tags"]; ok {
		json.Unmarshal(v, &merged.Tags)
	}
	if v, ok := patch["pinned"]; ok {
		json.Unmarshal(v, &merged.Pinned)
	}

	out, err := h.repo.Update(r.Context(), merged)
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

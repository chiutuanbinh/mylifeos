package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/chiutuanbinh/mylifeos/backend/internal/repo"
)

type EventHandler struct{ repo repo.EventRepo }

func NewEventHandler(r repo.EventRepo) *EventHandler { return &EventHandler{r} }

func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	q := r.URL.Query()
	events, err := h.repo.List(r.Context(), uid, q.Get("from"), q.Get("to"))
	if err != nil {
		log.Printf("events.List: %v", err)
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var e models.Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	e.UserID = uid
	if e.Color == "" {
		e.Color = "#1677ff"
	}
	out, err := h.repo.Create(r.Context(), e)
	if err != nil {
		log.Printf("events.Create: %v", err)
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(out)
}

func (h *EventHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var e models.Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	e.ID = chi.URLParam(r, "id")
	e.UserID = uid
	out, err := h.repo.Update(r.Context(), e)
	if err != nil {
		log.Printf("events.Update: %v", err)
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *EventHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	if err := h.repo.Delete(r.Context(), chi.URLParam(r, "id"), uid); err != nil {
		log.Printf("events.Delete: %v", err)
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.WriteHeader(204)
}

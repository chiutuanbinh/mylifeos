package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/chiutuanbinh/mylifeos/backend/internal/repo"
)

type GoalHandler struct{ repo repo.GoalRepo }

func NewGoalHandler(r repo.GoalRepo) *GoalHandler { return &GoalHandler{r} }

func (h *GoalHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	goals, err := h.repo.List(r.Context(), uid)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(goals)
}

func (h *GoalHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var g models.Goal
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	if g.Name == "" {
		http.Error(w, `{"error":"name is required"}`, 400)
		return
	}
	if len(g.Name) > 100 {
		http.Error(w, `{"error":"name too long"}`, 400)
		return
	}
	g.UserID = uid
	if g.Color == "" {
		g.Color = "#1677ff"
	}
	out, err := h.repo.Create(r.Context(), g)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(out)
}

func (h *GoalHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var g models.Goal
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	if g.Name == "" {
		http.Error(w, `{"error":"name is required"}`, 400)
		return
	}
	if len(g.Name) > 100 {
		http.Error(w, `{"error":"name too long"}`, 400)
		return
	}
	g.ID = chi.URLParam(r, "id")
	g.UserID = uid
	out, err := h.repo.Update(r.Context(), g)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *GoalHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	if err := h.repo.Delete(r.Context(), chi.URLParam(r, "id"), uid); err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.WriteHeader(204)
}

func (h *GoalHandler) AddKeyResult(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	goalID := chi.URLParam(r, "id")
	var body struct {
		Description  string  `json:"description"`
		Recurring    bool    `json:"recurring"`
		ReminderTime *string `json:"reminder_time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Description == "" {
		http.Error(w, `{"error":"description required"}`, 400)
		return
	}
	kr, err := h.repo.AddKeyResult(r.Context(), models.KeyResult{
		GoalID:       goalID,
		UserID:       uid,
		Description:  body.Description,
		Recurring:    body.Recurring,
		ReminderTime: body.ReminderTime,
	})
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(kr)
}

func (h *GoalHandler) UpdateKeyResult(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var body struct {
		Description  string  `json:"description"`
		Done         bool    `json:"done"`
		Recurring    bool    `json:"recurring"`
		ReminderTime *string `json:"reminder_time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	kr, err := h.repo.UpdateKeyResult(r.Context(), models.KeyResult{
		ID:           chi.URLParam(r, "kr_id"),
		UserID:       uid,
		Description:  body.Description,
		Done:         body.Done,
		Recurring:    body.Recurring,
		ReminderTime: body.ReminderTime,
	})
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(kr)
}

func (h *GoalHandler) DeleteKeyResult(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	if err := h.repo.DeleteKeyResult(r.Context(), chi.URLParam(r, "kr_id"), uid); err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.WriteHeader(204)
}

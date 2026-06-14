package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/chiutuanbinh/mylifeos/backend/internal/repo"
)

type HabitHandler struct{ repo repo.HabitRepo }

func NewHabitHandler(r repo.HabitRepo) *HabitHandler { return &HabitHandler{r} }

func (h *HabitHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	habits, err := h.repo.List(r.Context(), uid)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(habits)
}

func (h *HabitHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var habit models.Habit
	if err := json.NewDecoder(r.Body).Decode(&habit); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	habit.UserID = uid
	if habit.Icon == "" {
		habit.Icon = "✓"
	}
	out, err := h.repo.Create(r.Context(), habit)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(out)
}

func (h *HabitHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	if err := h.repo.Delete(r.Context(), chi.URLParam(r, "id"), uid); err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.WriteHeader(204)
}

func (h *HabitHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	q := r.URL.Query()
	from, to := q.Get("from"), q.Get("to")
	var (
		logs []models.HabitLog
		err  error
	)
	if from != "" && to != "" {
		logs, err = h.repo.GetLogsRange(r.Context(), uid, from, to)
	} else {
		logs, err = h.repo.GetLogs(r.Context(), uid, q.Get("date"))
	}
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (h *HabitHandler) ToggleLog(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var body struct {
		Date string `json:"date"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	log, err := h.repo.ToggleLog(r.Context(), chi.URLParam(r, "id"), uid, body.Date)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(log)
}

func (h *HabitHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var habit models.Habit
	if err := json.NewDecoder(r.Body).Decode(&habit); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	if habit.Name == "" {
		http.Error(w, `{"error":"name is required"}`, 400)
		return
	}
	if len(habit.Name) > 80 {
		http.Error(w, `{"error":"name too long"}`, 400)
		return
	}
	habit.ID = chi.URLParam(r, "id")
	habit.UserID = uid
	if habit.Icon == "" {
		habit.Icon = "✓"
	}
	out, err := h.repo.Update(r.Context(), habit)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *HabitHandler) GetLogRange(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		http.Error(w, `{"error":"from and to are required"}`, 400)
		return
	}
	logs, err := h.repo.GetLogRange(r.Context(), chi.URLParam(r, "id"), uid, from, to)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

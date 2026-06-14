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

type GoalHandler struct{ repo repo.GoalRepo }

func NewGoalHandler(r repo.GoalRepo) *GoalHandler { return &GoalHandler{r} }

func (h *GoalHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	goals, err := h.repo.List(r.Context(), uid)
	if err != nil {
		log.Printf("goals.List error: %v", err)
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
		log.Printf("goals.Create error: %v", err)
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
		log.Printf("goals.Update error: %v", err)
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *GoalHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	if err := h.repo.Delete(r.Context(), chi.URLParam(r, "id"), uid); err != nil {
		log.Printf("goals.Delete error: %v", err)
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.WriteHeader(204)
}

func (h *GoalHandler) AddKeyResult(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var kr models.KeyResult
	if err := json.NewDecoder(r.Body).Decode(&kr); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	kr.GoalID = chi.URLParam(r, "id")
	kr.UserID = uid
	out, err := h.repo.AddKeyResult(r.Context(), kr)
	if err != nil {
		log.Printf("goals.AddKeyResult error: %v", err)
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(out)
}

func (h *GoalHandler) UpdateKeyResult(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var kr models.KeyResult
	if err := json.NewDecoder(r.Body).Decode(&kr); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	kr.ID = chi.URLParam(r, "kr_id")
	kr.UserID = uid
	out, err := h.repo.UpdateKeyResult(r.Context(), kr)
	if err != nil {
		log.Printf("goals.UpdateKeyResult error: %v", err)
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *GoalHandler) DeleteKeyResult(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	if err := h.repo.DeleteKeyResult(r.Context(), chi.URLParam(r, "kr_id"), uid); err != nil {
		log.Printf("goals.DeleteKeyResult error: %v", err)
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.WriteHeader(204)
}

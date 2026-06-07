package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/repo"
)

type DashboardHandler struct{ repo repo.DashboardRepo }

func NewDashboardHandler(r repo.DashboardRepo) *DashboardHandler {
	return &DashboardHandler{r}
}

func (h *DashboardHandler) Summary(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	log.Printf("dashboard: summary request for uid=%q", uid)
	summary, err := h.repo.Summary(r.Context(), uid)
	if err != nil {
		log.Printf("dashboard: summary error: %v", err)
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

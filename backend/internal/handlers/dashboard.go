package handlers

import (
	"encoding/json"
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
	summary, err := h.repo.Summary(r.Context(), uid)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

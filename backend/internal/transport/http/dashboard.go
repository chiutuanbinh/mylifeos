package httphandler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	dashboardsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/dashboard"
)

type DashboardHandler struct{ svc *dashboardsvc.Service }

func NewDashboardHandler(svc *dashboardsvc.Service) *DashboardHandler {
	return &DashboardHandler{svc}
}

func (h *DashboardHandler) Summary(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	log.Printf("dashboard: summary request for uid=%q", uid)
	summary, err := h.svc.Summary(r.Context(), uid)
	if err != nil {
		log.Printf("dashboard: summary error: %v", err)
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

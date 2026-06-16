package httphandler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
)

type KRLogHandler struct{ repo repository.KRLogRepo }

func NewKRLogHandler(r repository.KRLogRepo) *KRLogHandler { return &KRLogHandler{r} }

func (h *KRLogHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	date := r.URL.Query().Get("date")
	logs, err := h.repo.GetLogs(r.Context(), uid, date)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (h *KRLogHandler) GetLogRange(w http.ResponseWriter, r *http.Request) {
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

func (h *KRLogHandler) ToggleLog(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var body struct {
		Date string `json:"date"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	logEntry, err := h.repo.ToggleLog(r.Context(), chi.URLParam(r, "id"), uid, body.Date)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logEntry)
}

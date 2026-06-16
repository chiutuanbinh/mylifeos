package httphandler

import (
	"encoding/json"
	"net/http"

	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	settingsdomain "github.com/chiutuanbinh/mylifeos/backend/internal/domain/settings"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
)

type SettingsHandler struct{ repo repository.SettingsRepo }

func NewSettingsHandler(r repository.SettingsRepo) *SettingsHandler { return &SettingsHandler{r} }

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	s, err := h.repo.Get(r.Context(), uid)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var s settingsdomain.UserSettings
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	s.UserID = uid
	out, err := h.repo.Upsert(r.Context(), s)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

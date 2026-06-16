package httphandler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/wealth"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
)

type AssetHandler struct{ repo repository.AssetRepo }

func NewAssetHandler(r repository.AssetRepo) *AssetHandler { return &AssetHandler{r} }

func (h *AssetHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	assets, err := h.repo.List(r.Context(), uid)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assets)
}

func validateAsset(a wealth.Asset) string {
	if a.Name == "" {
		return "name is required"
	}
	if a.Category == "" {
		return "category is required"
	}
	if a.PurchaseValue != nil && *a.PurchaseValue < 0 {
		return "purchase_value must be >= 0"
	}
	if a.DepreciationRate < 0 || a.DepreciationRate > 1 {
		return "depreciation_rate must be between 0 and 1"
	}
	return ""
}

func (h *AssetHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var a wealth.Asset
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	if msg := validateAsset(a); msg != "" {
		http.Error(w, `{"error":"`+msg+`"}`, 400)
		return
	}
	a.UserID = uid
	out, err := h.repo.Create(r.Context(), a)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(out)
}

func (h *AssetHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var a wealth.Asset
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	if msg := validateAsset(a); msg != "" {
		http.Error(w, `{"error":"`+msg+`"}`, 400)
		return
	}
	a.ID = chi.URLParam(r, "id")
	a.UserID = uid
	out, err := h.repo.Update(r.Context(), a)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *AssetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	if err := h.repo.Delete(r.Context(), chi.URLParam(r, "id"), uid); err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.WriteHeader(204)
}

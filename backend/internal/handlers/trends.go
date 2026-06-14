package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/chiutuanbinh/mylifeos/backend/internal/repo"
	"github.com/chiutuanbinh/mylifeos/backend/internal/scraper"
)

type TrendsHandler struct {
	repo      repo.TrendsRepo
	assetRepo repo.AssetRepo
}

func NewTrendsHandler(r repo.TrendsRepo, a repo.AssetRepo) *TrendsHandler {
	return &TrendsHandler{repo: r, assetRepo: a}
}

func (h *TrendsHandler) ListSnapshots(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	snaps, err := h.repo.ListSnapshots(r.Context(), uid)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snaps)
}

func (h *TrendsHandler) AddSnapshot(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var body struct {
		Date     string  `json:"date"`
		NetWorth float64 `json:"net_worth"`
		Note     string  `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Date == "" {
		http.Error(w, `{"error":"date and net_worth required"}`, 400)
		return
	}
	snap, err := h.repo.UpsertSnapshot(r.Context(), models.NetWorthSnapshot{
		UserID:       uid,
		SnapshotDate: body.Date,
		NetWorth:     body.NetWorth,
		Note:         body.Note,
	})
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(snap)
}

func (h *TrendsHandler) ListBenchmarks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	sources := strings.Split(q.Get("sources"), ",")
	from := q.Get("from")
	to := q.Get("to")
	if from == "" {
		from = time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	}
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}
	var filteredSources []string
	for _, s := range sources {
		if s != "" {
			filteredSources = append(filteredSources, s)
		}
	}
	data, err := h.repo.ListBenchmarks(r.Context(), filteredSources, from, to)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *TrendsHandler) ListBankRates(w http.ResponseWriter, r *http.Request) {
	rates, err := h.repo.LatestBankRates(r.Context())
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rates)
}

func (h *TrendsHandler) ListNews(w http.ResponseWriter, r *http.Request) {
	news, err := h.repo.ListNews(r.Context(), 20)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(news)
}

// TriggerScrape manually triggers the scraper (non-blocking).
func (h *TrendsHandler) TriggerScrape(w http.ResponseWriter, r *http.Request) {
	go scraper.Run(r.Context(), h.repo)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "scrape started"})
}

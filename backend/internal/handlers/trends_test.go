package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/handlers"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
)

type mockTrendsRepo struct{ failAll bool }

type errTrendsRepo struct{}

func (m *errTrendsRepo) ListSnapshots(_ context.Context, _ string) ([]models.NetWorthSnapshot, error) {
	return nil, fmt.Errorf("db error")
}
func (m *errTrendsRepo) UpsertSnapshot(_ context.Context, s models.NetWorthSnapshot) (models.NetWorthSnapshot, error) {
	return models.NetWorthSnapshot{}, fmt.Errorf("db error")
}
func (m *errTrendsRepo) UpsertBenchmark(_ context.Context, _ models.BenchmarkData) error {
	return fmt.Errorf("db error")
}
func (m *errTrendsRepo) ListBenchmarks(_ context.Context, _ []string, _, _ string) ([]models.BenchmarkData, error) {
	return nil, fmt.Errorf("db error")
}
func (m *errTrendsRepo) LatestBankRates(_ context.Context) ([]models.BankRate, error) {
	return nil, fmt.Errorf("db error")
}
func (m *errTrendsRepo) UpsertBankRate(_ context.Context, _ models.BankRate) error {
	return fmt.Errorf("db error")
}
func (m *errTrendsRepo) ListNews(_ context.Context, _ int) ([]models.NewsItem, error) {
	return nil, fmt.Errorf("db error")
}
func (m *errTrendsRepo) UpsertNews(_ context.Context, _ []models.NewsItem) error {
	return fmt.Errorf("db error")
}



func (m *mockTrendsRepo) ListSnapshots(_ context.Context, _ string) ([]models.NetWorthSnapshot, error) {
	return []models.NetWorthSnapshot{
		{ID: "s-1", SnapshotDate: "2026-06-14", NetWorth: 500000000, Note: ""},
	}, nil
}
func (m *mockTrendsRepo) UpsertSnapshot(_ context.Context, s models.NetWorthSnapshot) (models.NetWorthSnapshot, error) {
	s.ID = "s-new"
	return s, nil
}
func (m *mockTrendsRepo) UpsertBenchmark(_ context.Context, _ models.BenchmarkData) error { return nil }
func (m *mockTrendsRepo) ListBenchmarks(_ context.Context, _ []string, _, _ string) ([]models.BenchmarkData, error) {
	return []models.BenchmarkData{
		{ID: "b-1", Source: "vn_index", Date: "2026-06-14", Value: 1250.5},
	}, nil
}
func (m *mockTrendsRepo) LatestBankRates(_ context.Context) ([]models.BankRate, error) {
	return []models.BankRate{
		{Bank: "vcb", Saving12m: 5.5, Lending: 9.0, FetchedDate: "2026-06-14"},
	}, nil
}
func (m *mockTrendsRepo) UpsertBankRate(_ context.Context, _ models.BankRate) error { return nil }
func (m *mockTrendsRepo) ListNews(_ context.Context, _ int) ([]models.NewsItem, error) {
	return []models.NewsItem{
		{ID: "n-1", Source: "vneconomy", Title: "VN-Index tăng mạnh", URL: "https://vneconomy.vn/1"},
	}, nil
}
func (m *mockTrendsRepo) UpsertNews(_ context.Context, _ []models.NewsItem) error { return nil }

func TestTrendsListSnapshots(t *testing.T) {
	devEnv(t)
	h := handlers.NewTrendsHandler(&mockTrendsRepo{}, &mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListSnapshots))

	req := httptest.NewRequest("GET", "/api/v1/net-worth-snapshots", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var snaps []models.NetWorthSnapshot
	if err := json.NewDecoder(w.Body).Decode(&snaps); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(snaps) != 1 || snaps[0].ID != "s-1" {
		t.Fatalf("unexpected: %+v", snaps)
	}
}

func TestTrendsAddSnapshot(t *testing.T) {
	devEnv(t)
	h := handlers.NewTrendsHandler(&mockTrendsRepo{}, &mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.AddSnapshot))

	body, _ := json.Marshal(map[string]any{"date": "2026-01-01", "net_worth": 400000000, "note": "backfill"})
	req := httptest.NewRequest("POST", "/api/v1/net-worth-snapshots", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTrendsAddSnapshot_MissingDate(t *testing.T) {
	devEnv(t)
	h := handlers.NewTrendsHandler(&mockTrendsRepo{}, &mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.AddSnapshot))

	body, _ := json.Marshal(map[string]any{"net_worth": 400000000})
	req := httptest.NewRequest("POST", "/api/v1/net-worth-snapshots", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTrendsListBenchmarks(t *testing.T) {
	devEnv(t)
	h := handlers.NewTrendsHandler(&mockTrendsRepo{}, &mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListBenchmarks))

	req := httptest.NewRequest("GET", "/api/v1/benchmarks?sources=vn_index&from=2026-01-01&to=2026-06-14", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var data []models.BenchmarkData
	json.NewDecoder(w.Body).Decode(&data)
	if len(data) != 1 || data[0].Source != "vn_index" {
		t.Fatalf("unexpected: %+v", data)
	}
}

func TestTrendsListBankRates(t *testing.T) {
	devEnv(t)
	h := handlers.NewTrendsHandler(&mockTrendsRepo{}, &mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListBankRates))

	req := httptest.NewRequest("GET", "/api/v1/bank-rates", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var rates []models.BankRate
	json.NewDecoder(w.Body).Decode(&rates)
	if len(rates) != 1 || rates[0].Bank != "vcb" {
		t.Fatalf("unexpected: %+v", rates)
	}
}

func TestTrendsListSnapshots_Error(t *testing.T) {
	devEnv(t)
	h := handlers.NewTrendsHandler(&errTrendsRepo{}, &mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListSnapshots))

	req := httptest.NewRequest("GET", "/api/v1/net-worth-snapshots", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestTrendsListBenchmarks_Error(t *testing.T) {
	devEnv(t)
	h := handlers.NewTrendsHandler(&errTrendsRepo{}, &mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListBenchmarks))

	req := httptest.NewRequest("GET", "/api/v1/benchmarks", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestTrendsListBankRates_Error(t *testing.T) {
	devEnv(t)
	h := handlers.NewTrendsHandler(&errTrendsRepo{}, &mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListBankRates))

	req := httptest.NewRequest("GET", "/api/v1/bank-rates", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestTrendsListNews_Error(t *testing.T) {
	devEnv(t)
	h := handlers.NewTrendsHandler(&errTrendsRepo{}, &mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListNews))

	req := httptest.NewRequest("GET", "/api/v1/news", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestTrendsTriggerScrape(t *testing.T) {
	devEnv(t)
	h := handlers.NewTrendsHandler(&mockTrendsRepo{}, &mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.TriggerScrape))

	req := httptest.NewRequest("POST", "/api/v1/scrape", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestTrendsAddSnapshot_Error(t *testing.T) {
	devEnv(t)
	h := handlers.NewTrendsHandler(&errTrendsRepo{}, &mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.AddSnapshot))

	body, _ := json.Marshal(map[string]any{"date": "2026-01-01", "net_worth": 400000000})
	req := httptest.NewRequest("POST", "/api/v1/net-worth-snapshots", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestTrendsListNews(t *testing.T) {
	devEnv(t)
	h := handlers.NewTrendsHandler(&mockTrendsRepo{}, &mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListNews))

	req := httptest.NewRequest("GET", "/api/v1/news", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var news []models.NewsItem
	json.NewDecoder(w.Body).Decode(&news)
	if len(news) != 1 || news[0].Source != "vneconomy" {
		t.Fatalf("unexpected: %+v", news)
	}
}

package httphandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	httphandler "github.com/chiutuanbinh/mylifeos/backend/internal/transport/http"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	trendsdomain "github.com/chiutuanbinh/mylifeos/backend/internal/domain/trends"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
)

// mockTrendsRepo implements repository.TrendsRepo using domain types.
type mockTrendsRepo struct{}

func (m *mockTrendsRepo) ListSnapshots(_ context.Context, _ string) ([]trendsdomain.NetWorthSnapshot, error) {
	return []trendsdomain.NetWorthSnapshot{
		{ID: "s-1", SnapshotDate: "2026-06-14", NetWorth: 500000000},
	}, nil
}
func (m *mockTrendsRepo) UpsertSnapshot(_ context.Context, s trendsdomain.NetWorthSnapshot) (trendsdomain.NetWorthSnapshot, error) {
	s.ID = "s-new"
	return s, nil
}
func (m *mockTrendsRepo) UpsertBenchmark(_ context.Context, _ trendsdomain.BenchmarkData) error {
	return nil
}
func (m *mockTrendsRepo) ListBenchmarks(_ context.Context, _ []string, _, _ string) ([]trendsdomain.BenchmarkData, error) {
	return []trendsdomain.BenchmarkData{
		{ID: "b-1", Source: "vn_index", Date: "2026-06-14", Value: 1250.5},
	}, nil
}
func (m *mockTrendsRepo) LatestBankRates(_ context.Context) ([]trendsdomain.BankRate, error) {
	return []trendsdomain.BankRate{
		{Bank: "vcb", Saving12m: 5.5, Lending: 9.0, FetchedDate: "2026-06-14"},
	}, nil
}
func (m *mockTrendsRepo) UpsertBankRate(_ context.Context, _ trendsdomain.BankRate) error {
	return nil
}
func (m *mockTrendsRepo) ListNews(_ context.Context, _ int) ([]trendsdomain.NewsItem, error) {
	return []trendsdomain.NewsItem{
		{ID: "n-1", Source: "vneconomy", Title: "VN-Index tăng mạnh", URL: "https://vneconomy.vn/1"},
	}, nil
}
func (m *mockTrendsRepo) UpsertNews(_ context.Context, _ []trendsdomain.NewsItem) error { return nil }

// errTrendsRepo returns errors for all calls (implements repository.TrendsRepo).
type errTrendsRepo struct{}

func (m *errTrendsRepo) ListSnapshots(_ context.Context, _ string) ([]trendsdomain.NetWorthSnapshot, error) {
	return nil, errors.New("db error")
}
func (m *errTrendsRepo) UpsertSnapshot(_ context.Context, s trendsdomain.NetWorthSnapshot) (trendsdomain.NetWorthSnapshot, error) {
	return trendsdomain.NetWorthSnapshot{}, errors.New("db error")
}
func (m *errTrendsRepo) UpsertBenchmark(_ context.Context, _ trendsdomain.BenchmarkData) error {
	return errors.New("db error")
}
func (m *errTrendsRepo) ListBenchmarks(_ context.Context, _ []string, _, _ string) ([]trendsdomain.BenchmarkData, error) {
	return nil, errors.New("db error")
}
func (m *errTrendsRepo) LatestBankRates(_ context.Context) ([]trendsdomain.BankRate, error) {
	return nil, errors.New("db error")
}
func (m *errTrendsRepo) UpsertBankRate(_ context.Context, _ trendsdomain.BankRate) error {
	return errors.New("db error")
}
func (m *errTrendsRepo) ListNews(_ context.Context, _ int) ([]trendsdomain.NewsItem, error) {
	return nil, errors.New("db error")
}
func (m *errTrendsRepo) UpsertNews(_ context.Context, _ []trendsdomain.NewsItem) error {
	return errors.New("db error")
}

// mockScraperRepo implements repo.TrendsRepo (old interface using models types) for scraper.
type mockScraperRepo struct{}

func (m *mockScraperRepo) ListSnapshots(_ context.Context, _ string) ([]models.NetWorthSnapshot, error) {
	return nil, nil
}
func (m *mockScraperRepo) UpsertSnapshot(_ context.Context, s models.NetWorthSnapshot) (models.NetWorthSnapshot, error) {
	return s, nil
}
func (m *mockScraperRepo) UpsertBenchmark(_ context.Context, _ models.BenchmarkData) error {
	return nil
}
func (m *mockScraperRepo) ListBenchmarks(_ context.Context, _ []string, _, _ string) ([]models.BenchmarkData, error) {
	return nil, nil
}
func (m *mockScraperRepo) LatestBankRates(_ context.Context) ([]models.BankRate, error) {
	return nil, nil
}
func (m *mockScraperRepo) UpsertBankRate(_ context.Context, _ models.BankRate) error { return nil }
func (m *mockScraperRepo) ListNews(_ context.Context, _ int) ([]models.NewsItem, error) {
	return nil, nil
}
func (m *mockScraperRepo) UpsertNews(_ context.Context, _ []models.NewsItem) error { return nil }

func TestTrendsListSnapshots(t *testing.T) {
	devEnv(t)
	h := httphandler.NewTrendsHandler(&mockTrendsRepo{}, &mockAssetRepo{}, &mockScraperRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListSnapshots))

	req := httptest.NewRequest("GET", "/api/v1/net-worth-snapshots", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var snaps []trendsdomain.NetWorthSnapshot
	if err := json.NewDecoder(w.Body).Decode(&snaps); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(snaps) != 1 || snaps[0].ID != "s-1" {
		t.Fatalf("unexpected: %+v", snaps)
	}
}

func TestTrendsAddSnapshot(t *testing.T) {
	devEnv(t)
	h := httphandler.NewTrendsHandler(&mockTrendsRepo{}, &mockAssetRepo{}, &mockScraperRepo{})
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
	h := httphandler.NewTrendsHandler(&mockTrendsRepo{}, &mockAssetRepo{}, &mockScraperRepo{})
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
	h := httphandler.NewTrendsHandler(&mockTrendsRepo{}, &mockAssetRepo{}, &mockScraperRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListBenchmarks))

	req := httptest.NewRequest("GET", "/api/v1/benchmarks?sources=vn_index&from=2026-01-01&to=2026-06-14", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var data []trendsdomain.BenchmarkData
	json.NewDecoder(w.Body).Decode(&data)
	if len(data) != 1 || data[0].Source != "vn_index" {
		t.Fatalf("unexpected: %+v", data)
	}
}

func TestTrendsListBankRates(t *testing.T) {
	devEnv(t)
	h := httphandler.NewTrendsHandler(&mockTrendsRepo{}, &mockAssetRepo{}, &mockScraperRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListBankRates))

	req := httptest.NewRequest("GET", "/api/v1/bank-rates", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var rates []trendsdomain.BankRate
	json.NewDecoder(w.Body).Decode(&rates)
	if len(rates) != 1 || rates[0].Bank != "vcb" {
		t.Fatalf("unexpected: %+v", rates)
	}
}

func TestTrendsListSnapshots_Error(t *testing.T) {
	devEnv(t)
	h := httphandler.NewTrendsHandler(&errTrendsRepo{}, &mockAssetRepo{}, &mockScraperRepo{})
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
	h := httphandler.NewTrendsHandler(&errTrendsRepo{}, &mockAssetRepo{}, &mockScraperRepo{})
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
	h := httphandler.NewTrendsHandler(&errTrendsRepo{}, &mockAssetRepo{}, &mockScraperRepo{})
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
	h := httphandler.NewTrendsHandler(&errTrendsRepo{}, &mockAssetRepo{}, &mockScraperRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListNews))

	req := httptest.NewRequest("GET", "/api/v1/news", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestTrendsAddSnapshot_Error(t *testing.T) {
	devEnv(t)
	h := httphandler.NewTrendsHandler(&errTrendsRepo{}, &mockAssetRepo{}, &mockScraperRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.AddSnapshot))

	body, _ := json.Marshal(map[string]any{"date": "2026-01-01", "net_worth": 1000})
	req := httptest.NewRequest("POST", "/api/v1/net-worth-snapshots", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

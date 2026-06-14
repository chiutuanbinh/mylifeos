package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/handlers"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
)

// mockTrendsRepo implements repo.TrendsRepo for testing.
type mockTrendsRepo struct {
	failList      bool
	failUpsert    bool
	failBenchmark bool
	failBankRates bool
	failNews      bool
}

func (m *mockTrendsRepo) ListSnapshots(_ context.Context, _ string) ([]models.NetWorthSnapshot, error) {
	if m.failList {
		return nil, errors.New("db error")
	}
	return []models.NetWorthSnapshot{{ID: "s-1", SnapshotDate: "2025-01-01", NetWorth: 100000}}, nil
}

func (m *mockTrendsRepo) UpsertSnapshot(_ context.Context, s models.NetWorthSnapshot) (models.NetWorthSnapshot, error) {
	if m.failUpsert {
		return models.NetWorthSnapshot{}, errors.New("db error")
	}
	s.ID = "s-new"
	return s, nil
}

func (m *mockTrendsRepo) UpsertBenchmark(_ context.Context, _ models.BenchmarkData) error {
	return nil
}

func (m *mockTrendsRepo) ListBenchmarks(_ context.Context, _ []string, _, _ string) ([]models.BenchmarkData, error) {
	if m.failBenchmark {
		return nil, errors.New("db error")
	}
	return []models.BenchmarkData{{ID: "b-1", Source: "sp500", Date: "2025-01-01", Value: 4500}}, nil
}

func (m *mockTrendsRepo) LatestBankRates(_ context.Context) ([]models.BankRate, error) {
	if m.failBankRates {
		return nil, errors.New("db error")
	}
	return []models.BankRate{{Bank: "vcb", Saving12m: 5.5, Lending: 9.0, FetchedDate: "2025-01-01"}}, nil
}

func (m *mockTrendsRepo) UpsertBankRate(_ context.Context, _ models.BankRate) error { return nil }

func (m *mockTrendsRepo) ListNews(_ context.Context, _ int) ([]models.NewsItem, error) {
	if m.failNews {
		return nil, errors.New("db error")
	}
	return []models.NewsItem{{ID: "n-1", Source: "rss", Title: "Test News", URL: "http://example.com"}}, nil
}

func (m *mockTrendsRepo) UpsertNews(_ context.Context, _ []models.NewsItem) error { return nil }

func newTrendsHandler(repo *mockTrendsRepo) *handlers.TrendsHandler {
	return handlers.NewTrendsHandler(repo, &mockAssetRepo{})
}

func TestTrendsListSnapshots(t *testing.T) {
	devEnv(t)
	h := newTrendsHandler(&mockTrendsRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListSnapshots))

	req := httptest.NewRequest("GET", "/api/v1/trends/snapshots", nil)
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

func TestTrendsListSnapshots_Error(t *testing.T) {
	devEnv(t)
	h := newTrendsHandler(&mockTrendsRepo{failList: true})
	handler := middleware.Auth(http.HandlerFunc(h.ListSnapshots))

	req := httptest.NewRequest("GET", "/api/v1/trends/snapshots", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestTrendsAddSnapshot(t *testing.T) {
	devEnv(t)
	h := newTrendsHandler(&mockTrendsRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.AddSnapshot))

	body, _ := json.Marshal(map[string]any{"date": "2025-06-01", "net_worth": 150000.0, "note": "test"})
	req := httptest.NewRequest("POST", "/api/v1/trends/snapshots", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var snap models.NetWorthSnapshot
	if err := json.NewDecoder(w.Body).Decode(&snap); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if snap.ID != "s-new" {
		t.Fatalf("unexpected id: %s", snap.ID)
	}
}

func TestTrendsAddSnapshot_BadJSON(t *testing.T) {
	devEnv(t)
	h := newTrendsHandler(&mockTrendsRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.AddSnapshot))

	req := httptest.NewRequest("POST", "/api/v1/trends/snapshots", bytes.NewReader([]byte("notjson")))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTrendsAddSnapshot_MissingDate(t *testing.T) {
	devEnv(t)
	h := newTrendsHandler(&mockTrendsRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.AddSnapshot))

	body, _ := json.Marshal(map[string]any{"net_worth": 100000.0})
	req := httptest.NewRequest("POST", "/api/v1/trends/snapshots", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTrendsAddSnapshot_UpsertError(t *testing.T) {
	devEnv(t)
	h := newTrendsHandler(&mockTrendsRepo{failUpsert: true})
	handler := middleware.Auth(http.HandlerFunc(h.AddSnapshot))

	body, _ := json.Marshal(map[string]any{"date": "2025-06-01", "net_worth": 100000.0})
	req := httptest.NewRequest("POST", "/api/v1/trends/snapshots", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestTrendsListBenchmarks(t *testing.T) {
	devEnv(t)
	h := newTrendsHandler(&mockTrendsRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListBenchmarks))

	req := httptest.NewRequest("GET", "/api/v1/trends/benchmarks?sources=sp500&from=2024-01-01&to=2025-01-01", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var data []models.BenchmarkData
	if err := json.NewDecoder(w.Body).Decode(&data); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(data) != 1 {
		t.Fatalf("unexpected: %+v", data)
	}
}

func TestTrendsListBenchmarks_DefaultDates(t *testing.T) {
	devEnv(t)
	h := newTrendsHandler(&mockTrendsRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListBenchmarks))

	// No from/to — should use defaults
	req := httptest.NewRequest("GET", "/api/v1/trends/benchmarks", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestTrendsListBenchmarks_Error(t *testing.T) {
	devEnv(t)
	h := newTrendsHandler(&mockTrendsRepo{failBenchmark: true})
	handler := middleware.Auth(http.HandlerFunc(h.ListBenchmarks))

	req := httptest.NewRequest("GET", "/api/v1/trends/benchmarks", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestTrendsListBankRates(t *testing.T) {
	devEnv(t)
	h := newTrendsHandler(&mockTrendsRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListBankRates))

	req := httptest.NewRequest("GET", "/api/v1/trends/bank-rates", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var rates []models.BankRate
	if err := json.NewDecoder(w.Body).Decode(&rates); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rates) != 1 || rates[0].Bank != "vcb" {
		t.Fatalf("unexpected: %+v", rates)
	}
}

func TestTrendsListBankRates_Error(t *testing.T) {
	devEnv(t)
	h := newTrendsHandler(&mockTrendsRepo{failBankRates: true})
	handler := middleware.Auth(http.HandlerFunc(h.ListBankRates))

	req := httptest.NewRequest("GET", "/api/v1/trends/bank-rates", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestTrendsListNews(t *testing.T) {
	devEnv(t)
	h := newTrendsHandler(&mockTrendsRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.ListNews))

	req := httptest.NewRequest("GET", "/api/v1/trends/news", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var news []models.NewsItem
	if err := json.NewDecoder(w.Body).Decode(&news); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(news) != 1 {
		t.Fatalf("unexpected: %+v", news)
	}
}

func TestTrendsListNews_Error(t *testing.T) {
	devEnv(t)
	h := newTrendsHandler(&mockTrendsRepo{failNews: true})
	handler := middleware.Auth(http.HandlerFunc(h.ListNews))

	req := httptest.NewRequest("GET", "/api/v1/trends/news", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestTrendsTriggerScrape(t *testing.T) {
	devEnv(t)
	h := newTrendsHandler(&mockTrendsRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.TriggerScrape))

	req := httptest.NewRequest("POST", "/api/v1/trends/scrape", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "scrape started" {
		t.Fatalf("unexpected status: %s", resp["status"])
	}
}

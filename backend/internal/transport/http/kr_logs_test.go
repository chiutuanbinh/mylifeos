package httphandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	httphandler "github.com/chiutuanbinh/mylifeos/backend/internal/transport/http"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/goals"
)

type mockKRLogRepo struct{}

func (m *mockKRLogRepo) GetLogs(_ context.Context, _, _ string) ([]goals.KRLog, error) {
	return []goals.KRLog{{ID: "kl-1", KRID: "kr-1", Done: true, LoggedDate: "2026-06-14"}}, nil
}
func (m *mockKRLogRepo) GetLogRange(_ context.Context, _, _, _, _ string) ([]goals.KRLog, error) {
	return []goals.KRLog{}, nil
}
func (m *mockKRLogRepo) ToggleLog(_ context.Context, _, _, _ string) (goals.KRLog, error) {
	return goals.KRLog{ID: "kl-1", KRID: "kr-1", Done: true, LoggedDate: "2026-06-14"}, nil
}

func TestKRLogGetLogs(t *testing.T) {
	devEnv(t)
	h := httphandler.NewKRLogHandler(&mockKRLogRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.GetLogs))

	req := httptest.NewRequest("GET", "/api/v1/kr-logs?date=2026-06-14", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var logs []goals.KRLog
	if err := json.NewDecoder(w.Body).Decode(&logs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(logs) != 1 || logs[0].KRID != "kr-1" {
		t.Fatalf("unexpected: %+v", logs)
	}
}

func TestKRLogToggle(t *testing.T) {
	devEnv(t)
	h := httphandler.NewKRLogHandler(&mockKRLogRepo{})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "kr-1")

	body, _ := json.Marshal(map[string]string{"date": "2026-06-14"})
	req := httptest.NewRequest("POST", "/api/v1/key-results/kr-1/log", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := middleware.Auth(http.HandlerFunc(h.ToggleLog))
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var logEntry goals.KRLog
	if err := json.NewDecoder(w.Body).Decode(&logEntry); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !logEntry.Done {
		t.Fatalf("expected done=true")
	}
}

func TestKRLogGetLogRange(t *testing.T) {
	devEnv(t)
	h := httphandler.NewKRLogHandler(&mockKRLogRepo{})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "kr-1")

	req := httptest.NewRequest("GET", "/api/v1/key-results/kr-1/logs?from=2026-06-01&to=2026-06-14", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler := middleware.Auth(http.HandlerFunc(h.GetLogRange))
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestKRLogGetLogRange_MissingParams(t *testing.T) {
	devEnv(t)
	h := httphandler.NewKRLogHandler(&mockKRLogRepo{})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "kr-1")

	req := httptest.NewRequest("GET", "/api/v1/key-results/kr-1/logs", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler := middleware.Auth(http.HandlerFunc(h.GetLogRange))
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

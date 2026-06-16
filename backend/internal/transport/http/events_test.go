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
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/calendar"
)

type mockEventRepo struct{}

func (m *mockEventRepo) List(_ context.Context, _, _, _ string) ([]calendar.Event, error) {
	return []calendar.Event{{
		ID:      "evt-1",
		UserID:  "user-1",
		Title:   "Test",
		StartAt: "2026-06-13T09:00:00Z",
		EndAt:   "2026-06-13T10:00:00Z",
		Color:   "#1677ff",
		AllDay:  false,
	}}, nil
}

func (m *mockEventRepo) Create(_ context.Context, e calendar.Event) (calendar.Event, error) {
	e.ID = "evt-new"
	return e, nil
}

func (m *mockEventRepo) Update(_ context.Context, e calendar.Event) (calendar.Event, error) {
	return e, nil
}

func (m *mockEventRepo) Delete(_ context.Context, _, _ string) error { return nil }

func (m *mockEventRepo) UpsertFromGoogle(_ context.Context, _ string, events []calendar.Event) (int, error) {
	return len(events), nil
}

func TestEventList(t *testing.T) {
	devEnv(t)
	h := httphandler.NewEventHandler(&mockEventRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.List))

	req := httptest.NewRequest("GET", "/api/v1/events", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var evts []calendar.Event
	if err := json.NewDecoder(w.Body).Decode(&evts); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(evts) != 1 || evts[0].ID != "evt-1" {
		t.Fatalf("unexpected events: %+v", evts)
	}
}

func TestEventCreate(t *testing.T) {
	devEnv(t)
	h := httphandler.NewEventHandler(&mockEventRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(map[string]any{
		"title":    "New Event",
		"start_at": "2026-06-13T09:00:00Z",
		"end_at":   "2026-06-13T10:00:00Z",
		"color":    "#ff0000",
		"all_day":  false,
	})
	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var evt calendar.Event
	if err := json.NewDecoder(w.Body).Decode(&evt); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if evt.ID != "evt-new" {
		t.Fatalf("unexpected id: %s", evt.ID)
	}
}

func TestEventUpdate(t *testing.T) {
	devEnv(t)
	h := httphandler.NewEventHandler(&mockEventRepo{})

	r := chi.NewRouter()
	r.Put("/events/{id}", middleware.Auth(http.HandlerFunc(h.Update)).(http.HandlerFunc))

	body, _ := json.Marshal(map[string]any{"title": "Updated", "start_at": "2026-06-13T09:00:00Z", "end_at": "2026-06-13T10:00:00Z"})
	req := httptest.NewRequest("PUT", "/events/evt-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestEventUpdate_BadRequest(t *testing.T) {
	devEnv(t)
	h := httphandler.NewEventHandler(&mockEventRepo{})

	r := chi.NewRouter()
	r.Put("/events/{id}", middleware.Auth(http.HandlerFunc(h.Update)).(http.HandlerFunc))

	req := httptest.NewRequest("PUT", "/events/evt-1", bytes.NewReader([]byte("notjson")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestEventDelete(t *testing.T) {
	devEnv(t)
	h := httphandler.NewEventHandler(&mockEventRepo{})

	r := chi.NewRouter()
	r.Delete("/events/{id}", middleware.Auth(http.HandlerFunc(h.Delete)).(http.HandlerFunc))

	req := httptest.NewRequest("DELETE", "/events/evt-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

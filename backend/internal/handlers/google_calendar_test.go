package handlers

// White-box tests for google_calendar.go — access unexported functions directly.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
)

func TestMapGCalEvent_Timed(t *testing.T) {
	ge := gcalEvent{
		ID:      "evt-abc",
		Summary: "Team Meeting",
		Status:  "confirmed",
	}
	ge.Start.DateTime = "2026-06-13T09:00:00Z"
	ge.End.DateTime = "2026-06-13T10:00:00Z"

	e := mapGCalEvent(ge)

	if e.Title != "Team Meeting" {
		t.Errorf("title: got %q", e.Title)
	}
	if e.StartAt != "2026-06-13T09:00:00Z" {
		t.Errorf("start_at: got %q", e.StartAt)
	}
	if e.AllDay {
		t.Error("expected AllDay=false")
	}
	if e.GoogleEventID == nil || *e.GoogleEventID != "evt-abc" {
		t.Errorf("google_event_id: got %v", e.GoogleEventID)
	}
}

func TestMapGCalEvent_AllDay(t *testing.T) {
	ge := gcalEvent{ID: "evt-allday", Summary: "Holiday"}
	ge.Start.Date = "2026-06-13"
	ge.End.Date = "2026-06-14"

	e := mapGCalEvent(ge)

	if !e.AllDay {
		t.Error("expected AllDay=true")
	}
	if e.StartAt != "2026-06-13T00:00:00Z" {
		t.Errorf("start_at: got %q", e.StartAt)
	}
}

func TestMapGCalEvent_NoTitle(t *testing.T) {
	ge := gcalEvent{ID: "x"}
	ge.Start.DateTime = "2026-06-13T09:00:00Z"
	ge.End.DateTime = "2026-06-13T10:00:00Z"

	e := mapGCalEvent(ge)

	if e.Title != "(no title)" {
		t.Errorf("expected '(no title)', got %q", e.Title)
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusCreated, map[string]string{"key": "val"})

	if w.Code != http.StatusCreated {
		t.Errorf("status: got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type: got %q", ct)
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["key"] != "val" {
		t.Errorf("body: got %v", body)
	}
}

// mockGCalServer returns a test HTTP server mimicking the Google Calendar API.
type mockGCalEventRepo struct{ upserted int }

func (m *mockGCalEventRepo) List(_ context.Context, _, _, _ string) ([]models.Event, error) {
	return []models.Event{}, nil
}
func (m *mockGCalEventRepo) Create(_ context.Context, e models.Event) (models.Event, error) {
	e.ID = "new"
	return e, nil
}
func (m *mockGCalEventRepo) Update(_ context.Context, e models.Event) (models.Event, error) {
	return e, nil
}
func (m *mockGCalEventRepo) Delete(_ context.Context, _, _ string) error { return nil }
func (m *mockGCalEventRepo) UpsertFromGoogle(_ context.Context, _ string, evts []models.Event) (int, error) {
	m.upserted = len(evts)
	return len(evts), nil
}

func TestGCalSync_BadRequest(t *testing.T) {
	repo := &mockGCalEventRepo{}
	h := NewGoogleCalendarHandler(repo)

	// Missing provider_token → 400
	body := strings.NewReader(`{"time_min":"2026-06-01T00:00:00Z","time_max":"2026-06-30T23:59:59Z"}`)
	req := httptest.NewRequest("POST", "/sync", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Sync(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGCalSync_WithMockGoogleServer(t *testing.T) {
	// Spin up a fake Google Calendar API server.
	gcalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gcalListResponse{Items: []gcalEvent{
			{ID: "g1", Summary: "Test Event"},
		}})
	}))
	defer gcalServer.Close()

	// Patch gcalBaseURL for this test.
	orig := gcalBaseURL
	gcalBaseURL = gcalServer.URL
	defer func() { gcalBaseURL = orig }()

	repo := &mockGCalEventRepo{}
	h := NewGoogleCalendarHandler(repo)

	body := strings.NewReader(`{"provider_token":"tok","time_min":"2026-06-01T00:00:00Z","time_max":"2026-06-30T23:59:59Z"}`)
	req := httptest.NewRequest("POST", "/sync", body)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user-1")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.Sync(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if repo.upserted != 1 {
		t.Errorf("expected 1 upserted, got %d", repo.upserted)
	}
}

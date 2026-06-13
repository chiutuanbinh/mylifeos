package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/chiutuanbinh/mylifeos/backend/internal/repo"
)

type GoogleCalendarHandler struct{ repo repo.EventRepo }

func NewGoogleCalendarHandler(r repo.EventRepo) *GoogleCalendarHandler {
	return &GoogleCalendarHandler{r}
}

// gcalBaseURL is the Google Calendar API base. Overridable in tests.
var gcalBaseURL = "https://www.googleapis.com/calendar/v3"

type gcalSyncRequest struct {
	ProviderToken string `json:"provider_token"`
	TimeMin       string `json:"time_min"`
	TimeMax       string `json:"time_max"`
}

type gcalSyncResponse struct {
	Synced int    `json:"synced"`
	Error  string `json:"error,omitempty"`
}

// gcalEvent mirrors the fields we need from the Google Calendar API response.
type gcalEvent struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
	Start   struct {
		DateTime string `json:"dateTime"`
		Date     string `json:"date"`
	} `json:"start"`
	End struct {
		DateTime string `json:"dateTime"`
		Date     string `json:"date"`
	} `json:"end"`
	Status string `json:"status"`
}

type gcalListResponse struct {
	Items []gcalEvent `json:"items"`
}

func (h *GoogleCalendarHandler) Sync(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)

	var req gcalSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ProviderToken == "" {
		writeJSON(w, 400, gcalSyncResponse{Error: "provider_token required"})
		return
	}

	// Default to current month if not specified.
	now := time.Now()
	timeMin := req.TimeMin
	timeMax := req.TimeMax
	if timeMin == "" {
		timeMin = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	}
	if timeMax == "" {
		timeMax = time.Date(now.Year(), now.Month()+1, 0, 23, 59, 59, 0, time.UTC).Format(time.RFC3339)
	}

	gcalEvents, err := fetchGCalEvents(req.ProviderToken, timeMin, timeMax)
	if err != nil {
		writeJSON(w, 502, gcalSyncResponse{Error: fmt.Sprintf("google calendar fetch failed: %v", err)})
		return
	}

	var toUpsert []models.Event
	for _, ge := range gcalEvents {
		if ge.Status == "cancelled" {
			continue
		}
		e := mapGCalEvent(ge)
		toUpsert = append(toUpsert, e)
	}

	count, err := h.repo.UpsertFromGoogle(r.Context(), uid, toUpsert)
	if err != nil {
		writeJSON(w, 500, gcalSyncResponse{Error: "db upsert failed"})
		return
	}

	writeJSON(w, 200, gcalSyncResponse{Synced: count})
}

func fetchGCalEvents(token, timeMin, timeMax string) ([]gcalEvent, error) {
	params := url.Values{
		"timeMin":      {timeMin},
		"timeMax":      {timeMax},
		"singleEvents": {"true"},
		"orderBy":      {"startTime"},
		"maxResults":   {"500"},
	}
	apiURL := gcalBaseURL + "/calendars/primary/events?" + params.Encode()

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var result gcalListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Items, nil
}

func mapGCalEvent(ge gcalEvent) models.Event {
	allDay := ge.Start.DateTime == ""
	startAt := ge.Start.DateTime
	endAt := ge.End.DateTime
	if allDay {
		// all-day events use date-only strings; convert to RFC3339
		startAt = ge.Start.Date + "T00:00:00Z"
		endAt = ge.End.Date + "T23:59:59Z"
	}
	title := ge.Summary
	if title == "" {
		title = "(no title)"
	}
	gid := ge.ID
	return models.Event{
		Title:         title,
		StartAt:       startAt,
		EndAt:         endAt,
		Color:         "#1677ff",
		AllDay:        allDay,
		GoogleEventID: &gid,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

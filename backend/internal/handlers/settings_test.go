package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/handlers"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
)

type mockSettingsRepo struct{}

func (m *mockSettingsRepo) Get(_ context.Context, _ string) (models.UserSettings, error) {
	return models.UserSettings{UserID: "user-1"}, nil
}
func (m *mockSettingsRepo) Upsert(_ context.Context, s models.UserSettings) (models.UserSettings, error) {
	return s, nil
}

func TestSettingsGet(t *testing.T) {
	devEnv(t)
	h := handlers.NewSettingsHandler(&mockSettingsRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Get))

	req := httptest.NewRequest("GET", "/api/v1/settings", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var s models.UserSettings
	if err := json.NewDecoder(w.Body).Decode(&s); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if s.UserID != "user-1" {
		t.Fatalf("unexpected user_id: %s", s.UserID)
	}
}

func TestSettingsUpdate(t *testing.T) {
	devEnv(t)
	h := handlers.NewSettingsHandler(&mockSettingsRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Update))

	body, _ := json.Marshal(map[string]any{
		"notifications":   map[string]any{"email": true},
		"modules_enabled": map[string]any{"finance": true},
	})
	req := httptest.NewRequest("PUT", "/api/v1/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSettingsUpdate_BadRequest(t *testing.T) {
	devEnv(t)
	h := handlers.NewSettingsHandler(&mockSettingsRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Update))

	req := httptest.NewRequest("PUT", "/api/v1/settings", bytes.NewReader([]byte("notjson")))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

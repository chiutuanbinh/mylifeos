package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
)

func TestAuthBadToken_JWKSFetchFails(t *testing.T) {
	// Point SUPABASE_URL at a non-existent server so getKeyFunc errors → 500.
	os.Setenv("ENV", "production")
	os.Setenv("SUPABASE_URL", "http://127.0.0.1:1") // nothing listening
	defer os.Unsetenv("ENV")
	defer os.Unsetenv("SUPABASE_URL")

	// Reset cached keyfunc so this test's URL is used.
	middleware.ResetKeyFunc()

	handler := middleware.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer sometoken")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Either 500 (JWKS fetch failed) or 401 (token invalid) is acceptable —
	// the important thing is the handler was NOT called.
	if w.Code == http.StatusOK {
		t.Fatalf("expected non-200, got %d", w.Code)
	}
}

func TestAuthDevMode_DefaultUID(t *testing.T) {
	os.Setenv("ENV", "development")
	os.Unsetenv("DEV_USER_ID")
	defer os.Unsetenv("ENV")

	var gotUID string
	handler := middleware.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUID = middleware.GetUserID(r)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotUID != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("expected default dev UID, got %s", gotUID)
	}
}

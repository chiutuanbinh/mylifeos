package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
)

func TestAuthDevelopmentMode(t *testing.T) {
	os.Setenv("ENV", "development")
	os.Setenv("DEV_USER_ID", "test-user-123")
	defer os.Unsetenv("ENV")
	defer os.Unsetenv("DEV_USER_ID")

	var gotUID string
	handler := middleware.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUID = middleware.GetUserID(r)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotUID != "test-user-123" {
		t.Errorf("expected test-user-123, got %s", gotUID)
	}
}

func TestAuthMissingToken(t *testing.T) {
	os.Setenv("ENV", "production")
	defer os.Unsetenv("ENV")

	handler := middleware.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthKeyFuncCacheHit(t *testing.T) {
	// Prime the cache, then hit it a second time without reset.
	os.Setenv("ENV", "production")
	os.Setenv("SUPABASE_URL", "http://127.0.0.1:19999")
	defer os.Unsetenv("ENV")
	defer os.Unsetenv("SUPABASE_URL")
	middleware.ResetKeyFunc()
	defer middleware.ResetKeyFunc()

	handler := middleware.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	makeReq := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer sometoken")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}

	// First call loads keyfunc; second hits cache path.
	makeReq()
	w := makeReq()
	if w.Code == http.StatusOK {
		t.Errorf("expected non-200, got %d", w.Code)
	}
}

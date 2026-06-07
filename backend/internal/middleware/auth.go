package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const UserIDKey ctxKey = "userID"

var (
	jwksOnce  sync.Once
	jwksKeyFunc jwt.Keyfunc
	jwksErr   error
)

func getKeyFunc() (jwt.Keyfunc, error) {
	jwksOnce.Do(func() {
		supabaseURL := os.Getenv("SUPABASE_URL")
		if supabaseURL == "" {
			jwksErr = fmt.Errorf("SUPABASE_URL not set")
			return
		}
		jwksURL := supabaseURL + "/auth/v1/.well-known/jwks.json"
		k, err := keyfunc.NewDefault([]string{jwksURL})
		if err != nil {
			jwksErr = fmt.Errorf("jwks init: %w", err)
			return
		}
		jwksKeyFunc = k.Keyfunc
	})
	return jwksKeyFunc, jwksErr
}

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("ENV") == "development" {
			uid := os.Getenv("DEV_USER_ID")
			if uid == "" {
				uid = "00000000-0000-0000-0000-000000000001"
			}
			ctx := context.WithValue(r.Context(), UserIDKey, uid)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			http.Error(w, `{"error":"missing token"}`, http.StatusUnauthorized)
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")

		keyFunc, err := getKeyFunc()
		if err != nil {
			http.Error(w, `{"error":"auth config error"}`, http.StatusInternalServerError)
			return
		}

		token, err := jwt.Parse(tokenStr, keyFunc)
		if err != nil || !token.Valid {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, `{"error":"invalid claims"}`, http.StatusUnauthorized)
			return
		}
		uid, _ := claims["sub"].(string)
		ctx := context.WithValue(r.Context(), UserIDKey, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(r *http.Request) string {
	uid, _ := r.Context().Value(UserIDKey).(string)
	return uid
}

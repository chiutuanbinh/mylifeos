package middleware

import (
	"context"
	"log"
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
	jwksMu      sync.Mutex
	jwksKeyFunc jwt.Keyfunc
)

func getKeyFunc() (jwt.Keyfunc, error) {
	jwksMu.Lock()
	defer jwksMu.Unlock()
	if jwksKeyFunc != nil {
		return jwksKeyFunc, nil
	}
	supabaseURL := os.Getenv("SUPABASE_URL")
	jwksURL := supabaseURL + "/auth/v1/.well-known/jwks.json"
	log.Printf("auth: fetching JWKS from %s", jwksURL)
	k, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		log.Printf("auth: JWKS init error: %v", err)
		return nil, err
	}
	log.Printf("auth: JWKS loaded successfully")
	jwksKeyFunc = k.Keyfunc
	return jwksKeyFunc, nil
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
			log.Printf("auth: getKeyFunc error: %v", err)
			http.Error(w, `{"error":"auth config error"}`, http.StatusInternalServerError)
			return
		}

		token, err := jwt.Parse(tokenStr, keyFunc)
		if err != nil || !token.Valid {
			log.Printf("auth: token parse error: %v", err)
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

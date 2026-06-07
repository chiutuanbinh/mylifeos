package middleware

import (
	"context"
	"encoding/base64"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const UserIDKey ctxKey = "userID"

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
		secret := os.Getenv("SUPABASE_JWT_SECRET")
		// Supabase signing key is base64-encoded; decode before verifying.
		// Fall back to raw bytes if not valid base64 (e.g. plain string secrets).
		keyBytes, err := base64.StdEncoding.DecodeString(secret)
		if err != nil {
			keyBytes = []byte(secret)
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return keyBytes, nil
		})
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

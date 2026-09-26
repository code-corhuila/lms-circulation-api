package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type administratorIDKey struct{}

// RequireAuth validates the Bearer JWT on every request it wraps — NGINX does not do
// this (library-docs/05-architecture/decisions/records/ADR-003-nginx-reverse-proxy.md);
// library-api always validates its own tokens.
//
// The token is issued by the access module once HU-01 (login) is implemented; this
// middleware only verifies an already-issued token's signature and expiry.
func RequireAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeUnauthorized(w)
				return
			}

			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				writeUnauthorized(w)
				return
			}

			administratorID, _ := claims["sub"].(string)
			ctx := context.WithValue(r.Context(), administratorIDKey{}, administratorID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdministratorID retrieves the authenticated Administrator's ID from context.
func AdministratorID(ctx context.Context) string {
	id, _ := ctx.Value(administratorIDKey{}).(string)
	return id
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"UNAUTHORIZED","message":"Authentication token required"}`))
}

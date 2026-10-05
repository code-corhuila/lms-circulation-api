package middleware

import (
	"context"
	"crypto/rsa"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type administratorIDKey struct{}

// RequireAuth accepts exactly two algorithms, each with its own dedicated key
// — see lms-membership-api's identical middleware for the full rationale
// (rules/2-anexos/C-api-hexagonal.md, numeral 5.3.7): RS256 for a real
// Administrator session (lms-access-api's public key), HS256 for an internal
// service-to-service call (a secret never used for anything else). Branching
// on token.Method before returning key material is what prevents an RS256
// token's public key from being replayed as a forged HMAC secret.
func RequireAuth(publicKey *rsa.PublicKey, internalSecret string) func(http.Handler) http.Handler {
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
				switch t.Method.(type) {
				case *jwt.SigningMethodRSA:
					return publicKey, nil
				case *jwt.SigningMethodHMAC:
					return []byte(internalSecret), nil
				default:
					return nil, jwt.ErrTokenSignatureInvalid
				}
			}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg(), jwt.SigningMethodHS256.Alg()}))
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

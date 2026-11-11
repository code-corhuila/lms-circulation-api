package middleware

import (
	"context"
	"net/http"
)

// Authorizer is the driving-side view of iamclient.Client — an interface so
// this middleware can be unit-tested without a real lms-iam running
// (ADR-012-iam-service-scope.md).
type Authorizer interface {
	Authorize(ctx context.Context, administratorID, permission string) (bool, error)
}

// RequirePermission calls lms-iam's /authorize before letting the request
// through. Runs after RequireAuth, which already put the caller's sub into
// context. A caller lms-iam doesn't have an explicit role assignment for
// (including every internal service token, and every Administrator today)
// defaults to allowed — ADR-012's backward-compatible default — so wiring
// this in does not change today's real behavior until roles actually
// start getting assigned.
func RequirePermission(authorizer Authorizer, permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callerID := AdministratorID(r.Context())

			allowed, err := authorizer.Authorize(r.Context(), callerID, permission)
			if err != nil {
				writeAuthzFailure(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
				return
			}
			if !allowed {
				writeAuthzFailure(w, http.StatusForbidden, "FORBIDDEN", "This role does not grant the required permission")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeAuthzFailure(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + code + `","message":"` + message + `"}`))
}

package middleware

import "net/http"

// RequireCaller restricts a route to a static list of internal service
// names — the "sub" claim of an HS256 internal token (ADR-013), same
// pattern already used in lms-membership-api/lms-catalog-api. It runs after
// RequireAuth, which already validated the token and put its sub into
// context. A real Administrator's RS256 sub (their administrator ID) will
// never match one of these service names, so this also rejects a user
// session on these routes — correct, since they exist only for
// service-to-service calls (lms-workflow, lms-worker), never for a human.
func RequireCaller(allowed ...string) func(http.Handler) http.Handler {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		allowedSet[name] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			caller := AdministratorID(r.Context())
			if _, ok := allowedSet[caller]; !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"FORBIDDEN","message":"This route is not reachable by this caller"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

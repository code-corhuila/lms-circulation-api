// Package middleware holds cross-cutting HTTP middleware — see
// library-docs/05-architecture/cross-cutting.md.
package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const CorrelationIDKey contextKey = "correlationId"

const CorrelationIDHeader = "X-Correlation-Id"

// CorrelationID assigns a correlation ID to every request (reusing an inbound one from
// nginx if present) and propagates it via context and the response header, so it can be
// included in every log line for that request.
func CorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(CorrelationIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set(CorrelationIDHeader, id)
		ctx := context.WithValue(r.Context(), CorrelationIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// FromContext retrieves the correlation ID set by CorrelationID, or "" if absent.
func FromContext(ctx context.Context) string {
	id, _ := ctx.Value(CorrelationIDKey).(string)
	return id
}

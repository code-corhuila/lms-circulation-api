package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func withCaller(sub string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), administratorIDKey{}, sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequireCaller_Allowed(t *testing.T) {
	handler := withCaller("lms-workflow")(RequireCaller("lms-workflow")(okHandler()))

	req := httptest.NewRequest(http.MethodPost, "/loans/1/mark-late", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireCaller_SecondAllowedCaller(t *testing.T) {
	handler := withCaller("lms-worker")(RequireCaller("lms-workflow", "lms-worker")(okHandler()))

	req := httptest.NewRequest(http.MethodPost, "/loans/1/notify", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireCaller_Denied(t *testing.T) {
	handler := withCaller("catalog-service")(RequireCaller("lms-workflow")(okHandler()))

	req := httptest.NewRequest(http.MethodPost, "/loans/1/mark-late", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestRequireCaller_EmptySubDenied(t *testing.T) {
	handler := RequireCaller("lms-worker")(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/loans/accrue-fines", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

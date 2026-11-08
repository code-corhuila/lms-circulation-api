package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/code-corhuila/lms-circulation-api/internal/adapter/in/httpapi/middleware"
)

// fakeAuthorizer — Fake test double (11-quality/tdd-guide.md).
type fakeAuthorizer struct {
	allowed bool
	err     error
	gotPerm string
}

func (f *fakeAuthorizer) Authorize(_ context.Context, _, permission string) (bool, error) {
	f.gotPerm = permission
	return f.allowed, f.err
}

func TestRequirePermission_AllowsWhenAuthorized(t *testing.T) {
	fake := &fakeAuthorizer{allowed: true}
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })

	h := middleware.RequirePermission(fake, "loans:write")(next)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loans", nil)
	h.ServeHTTP(rec, req)

	if !called {
		t.Fatalf("expected the next handler to run")
	}
	if fake.gotPerm != "loans:write" {
		t.Fatalf("expected permission %q to reach the authorizer, got %q", "loans:write", fake.gotPerm)
	}
}

func TestRequirePermission_RejectsWhenNotAuthorized(t *testing.T) {
	fake := &fakeAuthorizer{allowed: false}
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })

	h := middleware.RequirePermission(fake, "loans:write")(next)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loans", nil)
	h.ServeHTTP(rec, req)

	if called {
		t.Fatalf("expected the next handler NOT to run")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestRequirePermission_FailsClosedOnAuthorizerError(t *testing.T) {
	fake := &fakeAuthorizer{err: errors.New("lms-iam unreachable")}
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })

	h := middleware.RequirePermission(fake, "loans:write")(next)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loans", nil)
	h.ServeHTTP(rec, req)

	if called {
		t.Fatalf("expected the next handler NOT to run when the authorizer call fails")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

package httpserver

import (
	"crypto/rsa"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/code-corhuila/lms-circulation-api/internal/adapter/in/httpapi/handler"
	"github.com/code-corhuila/lms-circulation-api/internal/adapter/in/httpapi/middleware"
)

// RouterConfig carries what the router needs to wire itself.
type RouterConfig struct {
	DB                *mongo.Client
	JWTPublicKey      *rsa.PublicKey
	InternalJWTSecret string
	CORSOrigin        string
	Loans             *handler.LoanHandler
	// Authorizer calls lms-iam's /authorize (ADR-012). nil disables the
	// check entirely (local dev without lms-iam running), never silently
	// passing a request through where a check was expected.
	Authorizer middleware.Authorizer
}

// NewRouter builds the chi router with the base middleware stack, health
// endpoints, and the Circulation bounded context's /loans routes.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.CorrelationID)
	r.Use(middleware.CORS(cfg.CORSOrigin))

	health := handler.NewHealthHandler(cfg.DB)
	r.Get("/health", health.Liveness)
	r.Get("/health/ready", health.Readiness)

	// requirePermission is a no-op when Authorizer is nil (lms-iam not
	// configured for this run) rather than a silent allow-everything.
	requirePermission := func(permission string) func(http.Handler) http.Handler {
		if cfg.Authorizer == nil {
			return func(next http.Handler) http.Handler { return next }
		}
		return middleware.RequirePermission(cfg.Authorizer, permission)
	}

	r.Route("/api/v1", func(api chi.Router) {
		api.Group(func(protected chi.Router) {
			protected.Use(middleware.RequireAuth(cfg.JWTPublicKey, cfg.InternalJWTSecret))

			protected.Route("/loans", func(loans chi.Router) {
				loans.With(requirePermission("loans:write")).Post("/", cfg.Loans.Create)           // HU-06
				loans.With(requirePermission("loans:read")).Get("/", cfg.Loans.List)               // HU-07
				loans.With(requirePermission("loans:read")).Get("/overdue", cfg.Loans.Overdue)     // HU-08, also polled by lms-workflow
				loans.With(requirePermission("loans:write")).Post("/{id}/return", cfg.Loans.Return) // HU-07 / HU-08 suspension trigger
			})
		})
	})

	return r
}

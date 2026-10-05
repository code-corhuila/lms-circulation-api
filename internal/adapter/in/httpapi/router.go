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

	r.Route("/api/v1", func(api chi.Router) {
		api.Group(func(protected chi.Router) {
			protected.Use(middleware.RequireAuth(cfg.JWTPublicKey, cfg.InternalJWTSecret))

			protected.Route("/loans", func(loans chi.Router) {
				loans.Post("/", cfg.Loans.Create)            // HU-06
				loans.Get("/", cfg.Loans.List)               // HU-07
				loans.Get("/overdue", cfg.Loans.Overdue)     // HU-08
				loans.Post("/{id}/return", cfg.Loans.Return) // HU-07 / HU-08 suspension trigger
			})
		})
	})

	return r
}

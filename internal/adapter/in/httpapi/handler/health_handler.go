// Package handler holds the primary (driving) HTTP adapters — one file per resource,
// added incrementally as each HU is implemented.
package handler

import (
	"context"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// HealthHandler implements the /health and /health/ready endpoints
// (library-docs/07-api/contracts/openapi/library-api.yaml).
type HealthHandler struct {
	DB *mongo.Client
}

func NewHealthHandler(db *mongo.Client) *HealthHandler {
	return &HealthHandler{DB: db}
}

// Liveness — GET /health — responds 200 if the process is alive.
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// Readiness — GET /health/ready — responds 200 only if the database is reachable.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	w.Header().Set("Content-Type", "application/json")
	if err := h.DB.Ping(ctx, nil); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"unavailable","database":"disconnected"}`))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","database":"connected"}`))
}

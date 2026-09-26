// Package persistence holds the secondary (driven) adapter that implements the Circulation
// bounded context's repository port against MongoDB — the one exception to "every
// service uses PostgreSQL" (library-docs/05-architecture/decisions/records/ADR-005-mongodb-for-circulation-service.md).
// A Loan document has no child tables and no FK to students/books, so it needs no
// relational feature Postgres would have given it. Relocated from
// internal/infrastructure/mongodb to match the folder name
// rules/2-anexos/C-api-hexagonal.md expects for outbound persistence adapters.
package persistence

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewClient connects to MongoDB and verifies connectivity with a bounded timeout —
// mirrors postgres.NewPool's shape in every other service.
func NewClient(ctx context.Context, uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connecting to mongo: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("pinging mongo: %w", err)
	}

	return client, nil
}

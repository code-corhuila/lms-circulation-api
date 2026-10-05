.PHONY: dev test test-cover build lint

dev:
	go run ./cmd/api/...

test:
	go test ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

build:
	go build -o bin/server ./cmd/api/...

lint:
	golangci-lint run ./...

# No migrate-up/migrate-down — this service persists to MongoDB (lms-circulation-db),
# not PostgreSQL, per ADR-005-mongodb-for-circulation-service.md. Indexes are created
# by the service itself at startup (see internal/infrastructure/mongodb).

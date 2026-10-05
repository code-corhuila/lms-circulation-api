# lms-circulation-api

> Circulation bounded context: loan, renewal, return, fines

Part of the **LMS Library** distributed system — team `lms-library`, Grupo 2.
Governance and documentation live in [`library-docs`](https://github.com/code-corhuila/library-docs).

Go REST API implementing the Circulation bounded context (`library-docs/02-domain/domain-map.md`)
— the **Core Domain**: loans, returns, and the suspension policy. Hexagonal Architecture — see
`library-docs/05-architecture/decisions/records/ADR-002-hexagonal-modular-monolith.md`.

This service owns the `loans` collection exclusively, in its own **MongoDB** database
(`loan_db`) — the one exception to "every service uses PostgreSQL", per
`ADR-005-mongodb-for-circulation-service.md`: a Loan document is flat with no FK to
students/books, so it gains nothing from a relational engine. It validates the JWT issued by
`lms-access-api` using the shared `JWT_SECRET`.

`LoanRegistrationService` coordinates Student (Membership), Book (Catalog), and Loan
(Circulation) over HTTP:

- `internal/adapter/out/membershipclient/client.go` → calls `lms-membership-api`
  (`GET /students/{id}` for eligibility, `POST /students/{id}/suspend` for late returns)
- `internal/adapter/out/catalogclient/client.go` → calls `lms-catalog-api`
  (`POST /books/{id}/loan-copy` / `/return-copy`)

Both clients authenticate with a short-lived internal token signed with the shared
`JWT_SECRET` (v1 has no separate service-to-service auth scope). There is no distributed
transaction across these calls — a mid-sequence failure is an accepted risk at this project's size.

This service combines Create (HU-06, "registrar préstamos"), Return + history/search (HU-07),
and the Overdue report with the late-return suspension trigger (HU-08). `SuspensionDays` is 3
(lowered from 7 per product decision).

Structure and contract follow `rules/2-anexos/C-api-hexagonal.md` (the course's own repository
norm) — see `ADR-010-liquibase-for-database-migrations.md` for the related `-db` decision.

## Structure

```
cmd/api/                       → entry point (main.go), the composition root
internal/
├── domain/circulation/          → Loan aggregate — no port, no framework import
├── application/
│   ├── service/                  → LoanRegistrationService (cross-service coordination) —
│   │                                 relocated from domain/service: it depends on ports, so
│   │                                 it's an application-layer concern, not domain
│   ├── port/
│   │   ├── in/                    → inbound ports the HTTP adapter depends on (loan_usecases.go)
│   │   └── out/                   → outbound ports (LoanRepository, StudentClient, BookClient,
│   │                                 IdempotencyStore) — ports.go
│   └── usecase/                   → RegisterLoan (HU-06), ReturnLoan, SearchLoans (HU-07),
│                                     OverdueLoans (HU-08)
├── config/                      → environment variable loading
├── adapter/
│   ├── in/httpapi/                → chi router, middleware, handlers, response envelope
│   └── out/
│       ├── persistence/             → LoanRepository and IdempotencyStore, both against MongoDB
│       ├── membershipclient/          → HTTP client for the Membership driven port
│       └── catalogclient/              → HTTP client for the Catalog driven port
└── infrastructure/logger/        → structured (zap) logger — not a port implementation
```

No `migrations/` — schema (the `loans` collection, its validator, its indexes) is owned by
`lms-circulation-db`'s own Liquibase migrations now, not this repo. This service no longer calls
`EnsureIndexes()` at startup; it assumes the collection and indexes already exist.

## Known gaps against `rules/2-anexos/C-api-hexagonal.md` / Anexo B

Both of this repo's previously-declared gaps are closed: idempotent creation is durable
(`internal/adapter/out/persistence.IdempotencyStore`, backed by `lms-circulation-db`'s
`idempotency_keys` collection), and auth is RS256/HS256 — see "Authentication" below.

## Tech Stack

* **Language:** Go 1.25
* **Router:** chi
* **Database driver:** `go.mongodb.org/mongo-driver` (MongoDB, schema now owned by
  `lms-circulation-db`'s Liquibase migrations)

## Authentication

Two algorithms, each with its own key (`rules/2-anexos/C-api-hexagonal.md`, numeral 5.3.7):
**RS256**, verified with `lms-access-api`'s public key, for a real Administrator session; **HS256**,
verified with a separate `INTERNAL_JWT_SECRET`, for the tokens this service mints to call
`lms-membership-api`/`lms-catalog-api` and for the ones it accepts from other services. Never the
same key for both — see `internal/adapter/in/httpapi/middleware/auth.go`'s doc comment for why
that specific separation is what prevents the RS256-to-HS256 key-confusion attack.

**`go.sum` is intentionally incomplete.** `mongo-driver` is a new dependency this project didn't
have before — its hashes (and its own transitive deps: bson, `xdg-go/scram`,
`xdg-go/stringprep`, `youmark/pkcs8`, `klauspost/compress`, ...) can't be generated without
running `go mod tidy` with a real Go toolchain and network access. Run `go mod tidy` once before
this builds; everything else in `go.sum` is already-verified hashes reused from
`lms-access-api`/`lms-membership-api`/`lms-catalog-api`.

## Development

```bash
go mod tidy   # required once — see the go.sum note above
go run ./cmd/api/...
```

Needs `membership-service` and `catalog-service` reachable at
`MEMBERSHIP_SERVICE_URL` / `CATALOG_SERVICE_URL` to register a loan or a return.

Standalone, against an already-running `lms-circulation-db`:

```bash
docker compose -f deploy/compose.yml up --build
```

Or, with the rest of the stack, from the repo root:

```bash
docker compose up --build circulation-service
```

## Tests

```bash
go test ./...
```

## Migration scope

**Comes from** [`lms-library-sandbox`](../lms-library-sandbox) (a local, unpublished prototype
repo, not `lms-library`) → `circulation-service/`: `cmd/`, `internal/{domain,application,infrastructure}`,
`Dockerfile`, `Makefile`. The web UI for loans (`pages/loans/`) already existed in `lms-library`
itself and is migrated separately to `lms-circulation-portal`.

**Corrected from Postgres to MongoDB.** The sandbox prototype persisted to PostgreSQL; this
migration rewrites only `internal/infrastructure/postgres` → `internal/infrastructure/mongodb`
(and `cmd/api/main.go`/`internal/config` to match) to comply with `ADR-005`, which was decided
after the prototype was built. Domain, application, and every other infrastructure adapter are
unchanged.

The full map lives in `library-docs`.

---

## Branching

Three permanent branches. **None of them accepts a direct commit** — you enter through a child
branch and leave through a Pull Request.

```
develop  <--PR--  feat/... fix/... chore/...
qa       <--PR--  qa/...
main     <--PR--  release/...  hotfix/...
```

Promotion happens **by re-application** (`git cherry-pick -x`), never by merging one permanent
branch into another: `merge develop -> qa` and `merge qa -> main` do not exist in this model.

`main` requires **1 approval from `ariel5253`**. On `develop` and `qa` the team sets its own review
rule.

Full policy: `00-governance/branching-policy.md` in `library-docs`.

## Correlations

* Domain rules → `library-docs/02-domain/entities-and-rules.md`
* Database engine decision → `ADR-005-mongodb-for-circulation-service.md`
* Database (own repo now) → [`lms-circulation-db`](https://github.com/code-corhuila/lms-circulation-db)
* API contract → `library-docs/07-api/contracts/openapi/library-api.yaml`
* Migration map / status → `library-docs/09-microservices/repo-migration-map.md`

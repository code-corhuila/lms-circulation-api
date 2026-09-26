// Package out holds the driven (outbound) ports every use case and domain
// service depends on — what the application needs from the outside world,
// never how it's implemented (rules/2-anexos/C-api-hexagonal.md, "Puertos de
// salida").
package out

import (
	"context"

	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
)

// LoanRepository is the driven port for Loan persistence.
type LoanRepository interface {
	FindByID(ctx context.Context, id string) (*circulation.Loan, error)
	CountActiveByStudent(ctx context.Context, studentID string) (int, error)
	Search(ctx context.Context, status string, overdueOnly bool, studentID, bookID string, page, limit int) (loans []*circulation.Loan, total int, err error)
	Save(ctx context.Context, l *circulation.Loan) error
}

// StudentClient is the driven port onto membership-service.
type StudentClient interface {
	IsEligible(ctx context.Context, studentID string) (bool, error)
	Suspend(ctx context.Context, studentID string, days int) error
}

// BookClient is the driven port onto catalog-service.
type BookClient interface {
	LoanCopy(ctx context.Context, bookID string) error
	ReturnCopy(ctx context.Context, bookID string) error
}

// IdempotencyStore is the driven port for the idempotent-creation check
// (rules/2-anexos/C-api-hexagonal.md, numeral 5.3.8): a repeated POST /loans
// with the same Idempotency-Key must return the original loan, not register a
// second one (and must not call catalog-service's loan-copy a second time).
//
// Provisional: the only adapter today (internal/adapter/out/idempotency) is an
// in-memory map — it does not survive a restart, and does not coordinate
// across more than one running instance. A durable adapter belongs in
// lms-circulation-db's own idempotency collection (rules/2-anexos/B-db-mongo.md),
// which does not exist yet.
type IdempotencyStore interface {
	// Get returns the loanID previously saved under key, and found=true, or
	// found=false if key has never been used.
	Get(ctx context.Context, key string) (loanID string, found bool, err error)
	// Save records that key produced loanID.
	Save(ctx context.Context, key, loanID string) error
}

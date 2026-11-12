// Package in holds the driving (inbound) ports — what this service offers,
// as interfaces the HTTP adapter depends on instead of the concrete use case
// structs directly (rules/2-anexos/C-api-hexagonal.md, "Puertos de entrada").
package in

import (
	"context"

	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
)

// RegisterLoanUseCase implements HU-06.
//
// replayed reports whether idempotencyKey had already been used: when true,
// the returned loan is the one originally created by that key, and the HTTP
// adapter must answer 200, not 201 (rules/2-anexos/C-api-hexagonal.md,
// numeral 5.3.8).
type RegisterLoanUseCase interface {
	Execute(ctx context.Context, studentID, bookID, idempotencyKey string) (loan *circulation.Loan, replayed bool, err error)
}

// ReturnLoanUseCase implements HU-07 (also triggers HU-08's suspension policy).
type ReturnLoanUseCase interface {
	Execute(ctx context.Context, loanID string) (*circulation.Loan, error)
}

// SearchLoansUseCase implements the history/query half of HU-07.
type SearchLoansUseCase interface {
	Execute(ctx context.Context, status string, overdueOnly bool, studentID, bookID string, page, limit int) (loans []*circulation.Loan, total int, err error)
}

// OverdueLoansUseCase implements HU-08's overdue report.
type OverdueLoansUseCase interface {
	Execute(ctx context.Context, page, limit int) (loans []*circulation.Loan, total int, err error)
}

// MarkLateUseCase implements ADR-008's step A: flag a loan as late once its
// due date has passed, called by lms-workflow's saga before it suspends the
// student in step B (lms-membership-api). Internal-only (ADR-013).
type MarkLateUseCase interface {
	Execute(ctx context.Context, loanID string) (*circulation.Loan, error)
}

// NotifyLoanUseCase records that a due-date notice was sent for a loan,
// called by lms-worker (ADR-009's first responsibility). Internal-only
// (ADR-013).
type NotifyLoanUseCase interface {
	Execute(ctx context.Context, loanID string) (*circulation.Loan, error)
}

// AccrueFinesUseCase is ADR-009's second responsibility. No fine/monetary
// penalty domain rule exists yet in this project
// (02-domain/entities-and-rules.md has none), so this is a deliberate stub
// — see usecase.AccrueFines. Internal-only (ADR-013).
type AccrueFinesUseCase interface {
	Execute(ctx context.Context) error
}

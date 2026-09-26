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

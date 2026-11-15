package usecase

import (
	"context"
	"time"

	out "github.com/code-corhuila/lms-circulation-api/internal/application/port/out"
	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
)

// MarkLate implements ADR-008's step A, called by lms-workflow's saga before
// it suspends the student in step B. Loan.MarkLate already makes a repeat
// call a no-op, so this use case doesn't need its own idempotency check on
// top of it.
type MarkLate struct {
	Loans out.LoanRepository
}

func NewMarkLate(loans out.LoanRepository) *MarkLate {
	return &MarkLate{Loans: loans}
}

func (uc *MarkLate) Execute(ctx context.Context, loanID string) (*circulation.Loan, error) {
	loan, err := uc.Loans.FindByID(ctx, loanID)
	if err != nil {
		return nil, err
	}
	if err := loan.MarkLate(time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := uc.Loans.Save(ctx, loan); err != nil {
		return nil, err
	}
	return loan, nil
}

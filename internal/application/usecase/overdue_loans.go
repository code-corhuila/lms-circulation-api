package usecase

import (
	"context"

	out "github.com/code-corhuila/lms-circulation-api/internal/application/port/out"
	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
)

// OverdueLoans implements HU-08's "Overdue Loans" report (Scenario 2): active
// loans whose due date has already passed.
type OverdueLoans struct {
	Loans out.LoanRepository
}

func NewOverdueLoans(loans out.LoanRepository) *OverdueLoans {
	return &OverdueLoans{Loans: loans}
}

func (uc *OverdueLoans) Execute(ctx context.Context, page, limit int) ([]*circulation.Loan, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return uc.Loans.Search(ctx, circulation.StatusActive, true, "", "", page, limit)
}

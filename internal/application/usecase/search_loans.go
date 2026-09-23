package usecase

import (
	"context"

	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
)

// SearchLoans implements the history/query half of HU-07 (and is reused by
// HU-08 with overdueOnly=true).
type SearchLoans struct {
	Loans circulation.LoanRepository
}

func NewSearchLoans(loans circulation.LoanRepository) *SearchLoans {
	return &SearchLoans{Loans: loans}
}

func (uc *SearchLoans) Execute(ctx context.Context, status string, overdueOnly bool, studentID, bookID string, page, limit int) ([]*circulation.Loan, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return uc.Loans.Search(ctx, status, overdueOnly, studentID, bookID, page, limit)
}

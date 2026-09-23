package usecase

import (
	"context"

	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
	"github.com/code-corhuila/lms-circulation-api/internal/domain/service"
)

// ReturnLoan implements HU-07's acceptance criteria — a thin wrapper around
// LoanRegistrationService.RegisterReturn, which also triggers the HU-08
// suspension policy when the return is late.
type ReturnLoan struct {
	service *service.LoanRegistrationService
}

func NewReturnLoan(svc *service.LoanRegistrationService) *ReturnLoan {
	return &ReturnLoan{service: svc}
}

func (uc *ReturnLoan) Execute(ctx context.Context, loanID string) (*circulation.Loan, error) {
	return uc.service.RegisterReturn(ctx, loanID)
}

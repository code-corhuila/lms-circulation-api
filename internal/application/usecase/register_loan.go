package usecase

import (
	"context"

	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
	"github.com/code-corhuila/lms-circulation-api/internal/domain/service"
)

// RegisterLoan implements HU-06's acceptance criteria. It's a thin wrapper
// around LoanRegistrationService — the actual cross-service coordination
// (Student eligibility, Book availability, Loan creation) lives in the
// domain service, per library-docs/02-domain/entities-and-rules.md
// ("Domain Services").
type RegisterLoan struct {
	service *service.LoanRegistrationService
}

func NewRegisterLoan(svc *service.LoanRegistrationService) *RegisterLoan {
	return &RegisterLoan{service: svc}
}

func (uc *RegisterLoan) Execute(ctx context.Context, studentID, bookID string) (*circulation.Loan, error) {
	return uc.service.RegisterLoan(ctx, studentID, bookID)
}

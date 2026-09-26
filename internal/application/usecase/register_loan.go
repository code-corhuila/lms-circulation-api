package usecase

import (
	"context"

	out "github.com/code-corhuila/lms-circulation-api/internal/application/port/out"
	"github.com/code-corhuila/lms-circulation-api/internal/application/service"
	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
)

// RegisterLoan implements HU-06's acceptance criteria, plus the idempotent
// creation rules/2-anexos/C-api-hexagonal.md (numeral 5.3.8) requires: a
// repeated request carrying the same Idempotency-Key must not register a
// second Loan (and must not call catalog-service's loan-copy a second time —
// LoanRegistrationService.RegisterLoan is only invoked when the key is new).
// The cross-service coordination itself (Student eligibility, Book
// availability, Loan creation) stays in LoanRegistrationService.
type RegisterLoan struct {
	service     *service.LoanRegistrationService
	loans       out.LoanRepository
	idempotency out.IdempotencyStore
}

func NewRegisterLoan(svc *service.LoanRegistrationService, loans out.LoanRepository, idempotency out.IdempotencyStore) *RegisterLoan {
	return &RegisterLoan{service: svc, loans: loans, idempotency: idempotency}
}

// Execute registers a loan. replayed=true means idempotencyKey had already
// been used — the returned loan is the original one, and the caller (the
// HTTP adapter) must answer 200, not 201.
func (uc *RegisterLoan) Execute(ctx context.Context, studentID, bookID, idempotencyKey string) (loan *circulation.Loan, replayed bool, err error) {
	if idempotencyKey != "" {
		if existingID, found, err := uc.idempotency.Get(ctx, idempotencyKey); err != nil {
			return nil, false, err
		} else if found {
			existing, err := uc.loans.FindByID(ctx, existingID)
			if err != nil {
				return nil, false, err
			}
			return existing, true, nil
		}
	}

	newLoan, err := uc.service.RegisterLoan(ctx, studentID, bookID)
	if err != nil {
		return nil, false, err
	}

	if idempotencyKey != "" {
		if err := uc.idempotency.Save(ctx, idempotencyKey, newLoan.ID); err != nil {
			return nil, false, err
		}
	}

	return newLoan, false, nil
}

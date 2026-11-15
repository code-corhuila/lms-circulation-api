package usecase

import (
	"context"
	"time"

	out "github.com/code-corhuila/lms-circulation-api/internal/application/port/out"
	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
)

// NotifyLoan records that lms-worker's due-date-notice job (ADR-009) sent a
// notice for this loan. There is no real notification channel in this
// system yet — "send" happens in lms-worker's own log line
// (DueDateNotices.Execute); this only persists the fact, so GET
// /loans/overdue has a real field to filter on once it's extended to skip
// already-notified loans (a separate, pre-existing gap lms-worker's own
// client comment already flags — not solved here).
type NotifyLoan struct {
	Loans out.LoanRepository
}

func NewNotifyLoan(loans out.LoanRepository) *NotifyLoan {
	return &NotifyLoan{Loans: loans}
}

func (uc *NotifyLoan) Execute(ctx context.Context, loanID string) (*circulation.Loan, error) {
	loan, err := uc.Loans.FindByID(ctx, loanID)
	if err != nil {
		return nil, err
	}
	loan.MarkNotified(time.Now().UTC())
	if err := uc.Loans.Save(ctx, loan); err != nil {
		return nil, err
	}
	return loan, nil
}

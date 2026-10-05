package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/code-corhuila/lms-circulation-api/internal/application/usecase"
	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
)

func TestOverdueLoans_IdentifiesLoansPastDueDate(t *testing.T) {
	// HU-08, Scenario 2: an active loan whose due date has passed appears in
	// the Overdue Loans report.
	_, loan, _, _, loans := setupLoanFixture(t)
	loan.DueDate = time.Now().UTC().Add(-1 * time.Hour) // force overdue
	loans.byID[loan.ID] = loan

	uc := usecase.NewOverdueLoans(loans)
	overdue, total, err := uc.Execute(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(overdue) != 1 || overdue[0].ID != loan.ID {
		t.Fatalf("expected the overdue loan to be reported, got %d results", len(overdue))
	}
}

func TestOverdueLoans_DoesNotReportLoansNotYetDue(t *testing.T) {
	_, loan, _, _, loans := setupLoanFixture(t)
	loan.DueDate = time.Now().UTC().Add(circulation.LoanPeriodDays * 24 * time.Hour) // still ahead
	loans.byID[loan.ID] = loan

	uc := usecase.NewOverdueLoans(loans)
	overdue, total, err := uc.Execute(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 || len(overdue) != 0 {
		t.Fatalf("expected no overdue loans, got %d", len(overdue))
	}
}

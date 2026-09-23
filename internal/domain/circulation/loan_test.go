package circulation_test

import (
	"testing"
	"time"

	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
)

func TestNewLoan_SetsDueDateSevenDaysAfterLoanDate(t *testing.T) {
	// INV-001 on Loan: dueDate = loanDate + 7 days, no renewals in v1.
	loan := circulation.NewLoan("student-1", "book-1")

	expected := loan.LoanDate.AddDate(0, 0, circulation.LoanPeriodDays)
	if !loan.DueDate.Equal(expected) {
		t.Fatalf("expected dueDate=%v, got %v", expected, loan.DueDate)
	}
	if loan.Status != circulation.StatusActive {
		t.Fatalf("expected status=ACTIVE, got %s", loan.Status)
	}
}

func TestRegisterReturn_OnTime(t *testing.T) {
	loan := circulation.NewLoan("student-1", "book-1")

	onTime := loan.DueDate.Add(-1 * time.Hour)
	wasLate, err := loan.RegisterReturn(onTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wasLate {
		t.Fatal("expected wasLate=false for an on-time return")
	}
	if loan.Status != circulation.StatusReturned {
		t.Fatalf("expected status=RETURNED, got %s", loan.Status)
	}
}

func TestRegisterReturn_Late(t *testing.T) {
	loan := circulation.NewLoan("student-1", "book-1")

	late := loan.DueDate.Add(1 * time.Hour)
	wasLate, err := loan.RegisterReturn(late)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !wasLate {
		t.Fatal("expected wasLate=true for a late return")
	}
}

func TestRegisterReturn_RejectsDoubleReturn(t *testing.T) {
	// INV-005: a loan can only be returned once.
	loan := circulation.NewLoan("student-1", "book-1")
	_, _ = loan.RegisterReturn(time.Now().UTC())

	if _, err := loan.RegisterReturn(time.Now().UTC()); err != circulation.ErrLoanAlreadyReturned {
		t.Fatalf("expected ErrLoanAlreadyReturned, got %v", err)
	}
}

func TestIsOverdue(t *testing.T) {
	loan := circulation.NewLoan("student-1", "book-1")
	loan.DueDate = time.Now().UTC().Add(-1 * time.Hour) // force overdue for the test

	if !loan.IsOverdue() {
		t.Fatal("expected loan to be overdue")
	}
}

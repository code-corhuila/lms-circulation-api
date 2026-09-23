package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/code-corhuila/lms-circulation-api/internal/application/usecase"
	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
	"github.com/code-corhuila/lms-circulation-api/internal/domain/service"
)

func setupLoanFixture(t *testing.T) (*service.LoanRegistrationService, *circulation.Loan, *fakeStudentClient, *fakeBookClient, *fakeLoanRepo) {
	t.Helper()
	students := &fakeStudentClient{eligible: map[string]bool{}}
	books := &fakeBookClient{availableCopies: map[string]int{"book-1": 1}}
	loans := &fakeLoanRepo{byID: map[string]*circulation.Loan{}, active: map[string]int{}}

	svc := service.NewLoanRegistrationService(students, books, loans)
	loan, err := svc.RegisterLoan(context.Background(), "student-1", "book-1")
	if err != nil {
		t.Fatalf("unexpected error setting up loan fixture: %v", err)
	}
	loans.byID[loan.ID] = loan
	return svc, loan, students, books, loans
}

func TestReturnLoan_OnTime(t *testing.T) {
	svc, loan, _, books, _ := setupLoanFixture(t)
	uc := usecase.NewReturnLoan(svc)

	returned, err := uc.Execute(context.Background(), loan.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if returned.Status != circulation.StatusReturned {
		t.Fatalf("expected status RETURNED, got %s", returned.Status)
	}
	if *returned.WasLate {
		t.Fatal("expected an on-time return")
	}
	if books.availableCopies[loan.BookID] != 1 {
		t.Fatalf("expected availableCopies restored to 1, got %d", books.availableCopies[loan.BookID])
	}
}

func TestReturnLoan_Late_SuspendsStudent(t *testing.T) {
	svc, loan, students, _, _ := setupLoanFixture(t)
	loan.DueDate = time.Now().UTC().Add(-1 * time.Hour) // force overdue for the test
	uc := usecase.NewReturnLoan(svc)

	returned, err := uc.Execute(context.Background(), loan.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !*returned.WasLate {
		t.Fatal("expected a late return")
	}
	if days := students.suspended[loan.StudentID]; days != circulation.SuspensionDays {
		t.Fatalf("expected the student suspended for %d days, got %d", circulation.SuspensionDays, days)
	}
}

func TestReturnLoan_RejectsDoubleReturn(t *testing.T) {
	svc, loan, _, _, _ := setupLoanFixture(t)
	uc := usecase.NewReturnLoan(svc)

	if _, err := uc.Execute(context.Background(), loan.ID); err != nil {
		t.Fatalf("unexpected error on first return: %v", err)
	}
	if _, err := uc.Execute(context.Background(), loan.ID); err != circulation.ErrLoanAlreadyReturned {
		t.Fatalf("expected ErrLoanAlreadyReturned, got %v", err)
	}
}

package usecase_test

import (
	"context"
	"testing"

	"github.com/code-corhuila/lms-circulation-api/internal/application/service"
	"github.com/code-corhuila/lms-circulation-api/internal/application/usecase"
	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
)

func TestRegisterLoan_Succeeds(t *testing.T) {
	students := &fakeStudentClient{eligible: map[string]bool{}}
	books := &fakeBookClient{availableCopies: map[string]int{"book-1": 1}}
	loans := &fakeLoanRepo{byID: map[string]*circulation.Loan{}, active: map[string]int{}}

	svc := service.NewLoanRegistrationService(students, books, loans)
	uc := usecase.NewRegisterLoan(svc, loans, newFakeIdempotencyStore())

	loan, replayed, err := uc.Execute(context.Background(), "student-1", "book-1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if replayed {
		t.Fatal("first registration must not be reported as replayed")
	}
	if loan.Status != circulation.StatusActive {
		t.Fatalf("expected status ACTIVE, got %s", loan.Status)
	}
	if books.availableCopies["book-1"] != 0 {
		t.Fatalf("expected availableCopies to be decremented to 0, got %d", books.availableCopies["book-1"])
	}
}

func TestRegisterLoan_RejectsSuspendedStudent(t *testing.T) {
	students := &fakeStudentClient{eligible: map[string]bool{"student-1": false}}
	books := &fakeBookClient{availableCopies: map[string]int{"book-1": 1}}
	loans := &fakeLoanRepo{byID: map[string]*circulation.Loan{}, active: map[string]int{}}

	svc := service.NewLoanRegistrationService(students, books, loans)
	uc := usecase.NewRegisterLoan(svc, loans, newFakeIdempotencyStore())

	_, _, err := uc.Execute(context.Background(), "student-1", "book-1", "")
	if err != service.ErrStudentSuspended {
		t.Fatalf("expected ErrStudentSuspended, got %v", err)
	}
}

func TestRegisterLoan_RejectsWhenLoanLimitReached(t *testing.T) {
	students := &fakeStudentClient{eligible: map[string]bool{}}
	books := &fakeBookClient{availableCopies: map[string]int{"book-1": 1}}
	loans := &fakeLoanRepo{byID: map[string]*circulation.Loan{}, active: map[string]int{"student-1": circulation.MaxActiveLoans}}

	svc := service.NewLoanRegistrationService(students, books, loans)
	uc := usecase.NewRegisterLoan(svc, loans, newFakeIdempotencyStore())

	_, _, err := uc.Execute(context.Background(), "student-1", "book-1", "")
	if err != service.ErrLoanLimitReached {
		t.Fatalf("expected ErrLoanLimitReached, got %v", err)
	}
}

func TestRegisterLoan_RejectsWhenNoCopiesAvailable(t *testing.T) {
	students := &fakeStudentClient{eligible: map[string]bool{}}
	books := &fakeBookClient{availableCopies: map[string]int{"book-1": 0}}
	loans := &fakeLoanRepo{byID: map[string]*circulation.Loan{}, active: map[string]int{}}

	svc := service.NewLoanRegistrationService(students, books, loans)
	uc := usecase.NewRegisterLoan(svc, loans, newFakeIdempotencyStore())

	_, _, err := uc.Execute(context.Background(), "student-1", "book-1", "")
	if err != (errNoCopiesAvailable{}) {
		t.Fatalf("expected no-copies error, got %v", err)
	}
}

func TestRegisterLoan_RepeatedIdempotencyKeyReplaysTheOriginal(t *testing.T) {
	// rules/2-anexos/C-api-hexagonal.md, numeral 5.3.8: a retry carrying the
	// same Idempotency-Key must return the original loan, not register a
	// second one (and must not decrement book availability twice).
	students := &fakeStudentClient{eligible: map[string]bool{}}
	books := &fakeBookClient{availableCopies: map[string]int{"book-1": 1}}
	loans := &fakeLoanRepo{byID: map[string]*circulation.Loan{}, active: map[string]int{}}

	svc := service.NewLoanRegistrationService(students, books, loans)
	uc := usecase.NewRegisterLoan(svc, loans, newFakeIdempotencyStore())

	first, replayed, err := uc.Execute(context.Background(), "student-1", "book-1", "retry-key-1")
	if err != nil {
		t.Fatalf("unexpected error on first registration: %v", err)
	}
	if replayed {
		t.Fatal("first registration must not be reported as replayed")
	}

	second, replayed, err := uc.Execute(context.Background(), "student-1", "book-1", "retry-key-1")
	if err != nil {
		t.Fatalf("unexpected error on retried registration: %v", err)
	}
	if !replayed {
		t.Fatal("expected the retry to be reported as replayed")
	}
	if second.ID != first.ID {
		t.Fatalf("expected the retry to return the original loan %s, got %s", first.ID, second.ID)
	}
	if books.availableCopies["book-1"] != 0 {
		t.Fatalf("expected availableCopies to be decremented only once, got %d", books.availableCopies["book-1"])
	}
}

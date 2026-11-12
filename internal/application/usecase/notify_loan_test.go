package usecase_test

import (
	"context"
	"testing"

	"github.com/code-corhuila/lms-circulation-api/internal/application/usecase"
	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
)

func TestNotifyLoan_SetsNotifiedAt(t *testing.T) {
	loan := circulation.NewLoan("student-1", "book-1")
	repo := &fakeLoanRepo{byID: map[string]*circulation.Loan{loan.ID: loan}}
	uc := usecase.NewNotifyLoan(repo)

	got, err := uc.Execute(context.Background(), loan.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.NotifiedAt == nil {
		t.Fatal("expected NotifiedAt to be set")
	}
}

func TestNotifyLoan_IsSafeToCallTwice(t *testing.T) {
	loan := circulation.NewLoan("student-1", "book-1")
	repo := &fakeLoanRepo{byID: map[string]*circulation.Loan{loan.ID: loan}}
	uc := usecase.NewNotifyLoan(repo)

	if _, err := uc.Execute(context.Background(), loan.ID); err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}
	if _, err := uc.Execute(context.Background(), loan.ID); err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
}

func TestNotifyLoan_NotFound(t *testing.T) {
	repo := &fakeLoanRepo{byID: map[string]*circulation.Loan{}}
	uc := usecase.NewNotifyLoan(repo)

	if _, err := uc.Execute(context.Background(), "missing"); err != circulation.ErrLoanNotFound {
		t.Fatalf("expected ErrLoanNotFound, got %v", err)
	}
}

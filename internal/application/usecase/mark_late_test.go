package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/code-corhuila/lms-circulation-api/internal/application/usecase"
	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
)

func TestMarkLate_FlagsAnActiveLoan(t *testing.T) {
	loan := circulation.NewLoan("student-1", "book-1")
	repo := &fakeLoanRepo{byID: map[string]*circulation.Loan{loan.ID: loan}}
	uc := usecase.NewMarkLate(repo)

	got, err := uc.Execute(context.Background(), loan.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.LateMarkedAt == nil {
		t.Fatal("expected LateMarkedAt to be set")
	}
}

func TestMarkLate_IsIdempotent(t *testing.T) {
	loan := circulation.NewLoan("student-1", "book-1")
	repo := &fakeLoanRepo{byID: map[string]*circulation.Loan{loan.ID: loan}}
	uc := usecase.NewMarkLate(repo)

	first, err := uc.Execute(context.Background(), loan.ID)
	if err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}
	firstMark := *first.LateMarkedAt

	time.Sleep(time.Millisecond)
	second, err := uc.Execute(context.Background(), loan.ID)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if !second.LateMarkedAt.Equal(firstMark) {
		t.Fatalf("expected the timestamp to stay %v, got %v", firstMark, *second.LateMarkedAt)
	}
}

func TestMarkLate_RejectsAReturnedLoan(t *testing.T) {
	loan := circulation.NewLoan("student-1", "book-1")
	_, _ = loan.RegisterReturn(time.Now().UTC())
	repo := &fakeLoanRepo{byID: map[string]*circulation.Loan{loan.ID: loan}}
	uc := usecase.NewMarkLate(repo)

	if _, err := uc.Execute(context.Background(), loan.ID); err != circulation.ErrLoanNotActive {
		t.Fatalf("expected ErrLoanNotActive, got %v", err)
	}
}

func TestMarkLate_NotFound(t *testing.T) {
	repo := &fakeLoanRepo{byID: map[string]*circulation.Loan{}}
	uc := usecase.NewMarkLate(repo)

	if _, err := uc.Execute(context.Background(), "missing"); err != circulation.ErrLoanNotFound {
		t.Fatalf("expected ErrLoanNotFound, got %v", err)
	}
}

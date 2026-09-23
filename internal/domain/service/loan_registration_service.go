// Package service holds Domain Services — business logic that coordinates more
// than one bounded context and therefore does not naturally belong inside a
// single Aggregate Root. See library-docs/02-domain/entities-and-rules.md,
// "Domain Services".
//
// Before the microservices split, StudentClient/BookClient were in-process
// repository calls against the same database. Now they're HTTP calls to
// membership-service and catalog-service — the coordination logic here is
// unchanged, only the driven ports' implementations moved to the network
// (library-docs/09-microservices/service-boundary-rules.md).
package service

import (
	"context"
	"errors"
	"time"

	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
)

// ErrStudentSuspended — INV-003 on Loan: a loan can only be created for a
// non-suspended student.
var ErrStudentSuspended = errors.New("student is suspended")

// ErrLoanLimitReached — INV-004 on Loan: max 2 simultaneous active loans per student.
var ErrLoanLimitReached = errors.New("loan limit reached")

// StudentClient is the driven port onto membership-service.
type StudentClient interface {
	IsEligible(ctx context.Context, studentID string) (bool, error)
	Suspend(ctx context.Context, studentID string, days int) error
}

// BookClient is the driven port onto catalog-service.
type BookClient interface {
	LoanCopy(ctx context.Context, bookID string) error
	ReturnCopy(ctx context.Context, bookID string) error
}

// LoanRegistrationService coordinates Student (Membership), Book (Catalog),
// and Loan (Circulation) — this is the one place in the codebase where those
// three are checked/changed together.
type LoanRegistrationService struct {
	students StudentClient
	books    BookClient
	loans    circulation.LoanRepository
}

func NewLoanRegistrationService(students StudentClient, books BookClient, loans circulation.LoanRepository) *LoanRegistrationService {
	return &LoanRegistrationService{students: students, books: books, loans: loans}
}

// RegisterLoan implements HU-06's acceptance criteria end to end.
func (s *LoanRegistrationService) RegisterLoan(ctx context.Context, studentID, bookID string) (*circulation.Loan, error) {
	eligible, err := s.students.IsEligible(ctx, studentID)
	if err != nil {
		return nil, err
	}
	if !eligible { // INV-003
		return nil, ErrStudentSuspended
	}

	activeCount, err := s.loans.CountActiveByStudent(ctx, studentID)
	if err != nil {
		return nil, err
	}
	if activeCount >= circulation.MaxActiveLoans { // INV-004
		return nil, ErrLoanLimitReached
	}

	if err := s.books.LoanCopy(ctx, bookID); err != nil { // INV-002 / INV-001 on Book
		return nil, err
	}

	loan := circulation.NewLoan(studentID, bookID) // INV-001 on Loan: dueDate = +7 days
	if err := s.loans.Save(ctx, loan); err != nil {
		return nil, err
	}

	return loan, nil
}

// RegisterReturn implements HU-07 and HU-08's acceptance criteria: closes the
// loan, restores availability, and — if the return was late — applies the
// flat suspension.
func (s *LoanRegistrationService) RegisterReturn(ctx context.Context, loanID string) (*circulation.Loan, error) {
	loan, err := s.loans.FindByID(ctx, loanID)
	if err != nil {
		return nil, err
	}

	returnTime := time.Now().UTC()
	wasLate, err := loan.RegisterReturn(returnTime) // INV-005
	if err != nil {
		return nil, err
	}

	if err := s.books.ReturnCopy(ctx, loan.BookID); err != nil {
		return nil, err
	}

	if wasLate { // INV-006 on Loan: late return triggers a fixed suspension
		if err := s.students.Suspend(ctx, loan.StudentID, circulation.SuspensionDays); err != nil {
			return nil, err
		}
	}

	if err := s.loans.Save(ctx, loan); err != nil {
		return nil, err
	}

	return loan, nil
}

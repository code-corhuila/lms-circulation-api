// Package circulation implements the Circulation bounded context — the Core Domain
// (library-docs/02-domain/domain-map.md): loans, returns, and the suspension policy.
package circulation

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// LoanPeriodDays and SuspensionDays are the confirmed v1 policy constants —
// library-docs/01-context/scope.md, Scope assumption #2, and
// library-docs/02-domain/entities-and-rules.md INV-001/INV-006 on Loan.
// SuspensionDays was lowered from 7 to 3 per product decision.
const (
	LoanPeriodDays = 7
	SuspensionDays = 3
	MaxActiveLoans = 2
)

// Status values for Loan.Status.
const (
	StatusActive   = "ACTIVE"
	StatusReturned = "RETURNED"
)

// ErrLoanAlreadyReturned — INV-005: a loan can only be returned once.
var ErrLoanAlreadyReturned = errors.New("loan already returned")

// ErrLoanNotFound is the port-level "not found" error.
var ErrLoanNotFound = errors.New("loan not found")

// Loan is the Circulation bounded context's Aggregate Root.
type Loan struct {
	ID         string
	StudentID  string
	BookID     string
	LoanDate   time.Time
	DueDate    time.Time
	ReturnDate *time.Time
	Status     string
	WasLate    *bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NewLoan creates a loan starting now, with dueDate fixed at LoanPeriodDays later —
// INV-001: dueDate is computed, never settable directly, and there is no renewal flow.
// Eligibility (available copy, non-suspended student, loan limit) must already have
// been checked by LoanRegistrationService before calling this constructor.
func NewLoan(studentID, bookID string) *Loan {
	now := time.Now().UTC()
	return &Loan{
		ID:        uuid.NewString(),
		StudentID: studentID,
		BookID:    bookID,
		LoanDate:  now,
		DueDate:   now.AddDate(0, 0, LoanPeriodDays),
		Status:    StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsOverdue returns true if the loan is still active and past its due date.
func (l *Loan) IsOverdue() bool {
	return l.Status == StatusActive && time.Now().UTC().After(l.DueDate)
}

// RegisterReturn closes the loan and reports whether it was late — the caller
// (LoanRegistrationService) is responsible for incrementing Book availability and,
// if late, applying the Student suspension (see library-docs/02-domain/domain-events.md,
// "Event flow: Return registration").
func (l *Loan) RegisterReturn(returnDate time.Time) (wasLate bool, err error) {
	if l.Status != StatusActive {
		return false, ErrLoanAlreadyReturned
	}
	wasLate = returnDate.After(l.DueDate)
	l.ReturnDate = &returnDate
	l.Status = StatusReturned
	l.WasLate = &wasLate
	l.UpdatedAt = time.Now().UTC()
	return wasLate, nil
}

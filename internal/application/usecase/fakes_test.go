package usecase_test

import (
	"context"
	"time"

	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
)

// --- Fakes (11-quality/tdd-guide.md) standing in for the HTTP calls to
// membership-service and catalog-service. Shared across this package's test
// files — do not redeclare in register_loan_test.go / return_loan_test.go.

type fakeStudentClient struct {
	eligible  map[string]bool
	suspended map[string]int
}

func (f *fakeStudentClient) IsEligible(_ context.Context, studentID string) (bool, error) {
	eligible, ok := f.eligible[studentID]
	if !ok {
		return true, nil
	}
	return eligible, nil
}
func (f *fakeStudentClient) Suspend(_ context.Context, studentID string, days int) error {
	if f.suspended == nil {
		f.suspended = map[string]int{}
	}
	f.suspended[studentID] = days
	f.eligible[studentID] = false
	return nil
}

type fakeBookClient struct {
	availableCopies map[string]int
}

func (f *fakeBookClient) LoanCopy(_ context.Context, bookID string) error {
	if f.availableCopies[bookID] <= 0 {
		return errNoCopiesAvailable{}
	}
	f.availableCopies[bookID]--
	return nil
}
func (f *fakeBookClient) ReturnCopy(_ context.Context, bookID string) error {
	f.availableCopies[bookID]++
	return nil
}

type errNoCopiesAvailable struct{}

func (errNoCopiesAvailable) Error() string { return "no copies available" }

type fakeLoanRepo struct {
	byID   map[string]*circulation.Loan
	active map[string]int
}

func (f *fakeLoanRepo) FindByID(_ context.Context, id string) (*circulation.Loan, error) {
	l, ok := f.byID[id]
	if !ok {
		return nil, circulation.ErrLoanNotFound
	}
	return l, nil
}
func (f *fakeLoanRepo) CountActiveByStudent(_ context.Context, studentID string) (int, error) {
	return f.active[studentID], nil
}
func (f *fakeLoanRepo) Search(_ context.Context, status string, overdueOnly bool, _, _ string, _, _ int) ([]*circulation.Loan, int, error) {
	var matches []*circulation.Loan
	for _, l := range f.byID {
		if status != "" && l.Status != status {
			continue
		}
		if overdueOnly && !(l.Status == circulation.StatusActive && time.Now().UTC().After(l.DueDate)) {
			continue
		}
		matches = append(matches, l)
	}
	return matches, len(matches), nil
}
func (f *fakeLoanRepo) Save(_ context.Context, l *circulation.Loan) error {
	f.byID[l.ID] = l
	return nil
}

type fakeIdempotencyStore struct {
	byKey map[string]string
}

func newFakeIdempotencyStore() *fakeIdempotencyStore {
	return &fakeIdempotencyStore{byKey: map[string]string{}}
}

func (f *fakeIdempotencyStore) Get(_ context.Context, key string) (string, bool, error) {
	id, ok := f.byKey[key]
	return id, ok, nil
}

func (f *fakeIdempotencyStore) Save(_ context.Context, key, loanID string) error {
	f.byKey[key] = loanID
	return nil
}

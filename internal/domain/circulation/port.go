package circulation

import "context"

// LoanRepository is the driven port for Loan persistence.
type LoanRepository interface {
	FindByID(ctx context.Context, id string) (*Loan, error)
	CountActiveByStudent(ctx context.Context, studentID string) (int, error)
	Search(ctx context.Context, status string, overdueOnly bool, studentID, bookID string, page, limit int) (loans []*Loan, total int, err error)
	Save(ctx context.Context, l *Loan) error
}

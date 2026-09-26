package persistence

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
)

const loansCollection = "loans"

// LoanRepository implements out.LoanRepository against MongoDB — the
// only code allowed to touch the `loans` collection
// (library-docs/09-microservices/service-boundary-rules.md).
//
// EnsureIndexes below still creates this collection's indexes from inside
// this -api repo, which rules/2-anexos/B-db-mongo.md says should live in
// lms-circulation-db instead ("Qué va en cada parte" — schema ownership never
// belongs to the -api). Not fixed in this change: ADR-010-liquibase-for-database-migrations.md
// tracks the -db restructuring this migration depends on; moving index
// creation out of here is a follow-up once that lands, not before.
type LoanRepository struct {
	collection *mongo.Collection
}

func NewLoanRepository(db *mongo.Database) *LoanRepository {
	return &LoanRepository{collection: db.Collection(loansCollection)}
}

// EnsureIndexes creates the indexes this repository relies on — MongoDB has no
// migration tool equivalent to golang-migrate, so schema evolution here is just
// "call this again with the new index definitions" (ADR-005's lighter-weight note).
// Called once at startup (see cmd/api/main.go), not per-request.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection(loansCollection).Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "student_id", Value: 1}}},
		{Keys: bson.D{{Key: "book_id", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "due_date", Value: 1}}},
	})
	return err
}

// loanDocument is the on-disk shape — kept separate from circulation.Loan so the
// domain aggregate carries no persistence-framework tags.
type loanDocument struct {
	ID         string     `bson:"_id"`
	StudentID  string     `bson:"student_id"`
	BookID     string     `bson:"book_id"`
	LoanDate   time.Time  `bson:"loan_date"`
	DueDate    time.Time  `bson:"due_date"`
	ReturnDate *time.Time `bson:"return_date,omitempty"`
	Status     string     `bson:"status"`
	WasLate    *bool      `bson:"was_late,omitempty"`
	CreatedAt  time.Time  `bson:"created_at"`
	UpdatedAt  time.Time  `bson:"updated_at"`
}

func toDocument(l *circulation.Loan) loanDocument {
	return loanDocument{
		ID:         l.ID,
		StudentID:  l.StudentID,
		BookID:     l.BookID,
		LoanDate:   l.LoanDate,
		DueDate:    l.DueDate,
		ReturnDate: l.ReturnDate,
		Status:     l.Status,
		WasLate:    l.WasLate,
		CreatedAt:  l.CreatedAt,
		UpdatedAt:  l.UpdatedAt,
	}
}

func (d loanDocument) toDomain() *circulation.Loan {
	return &circulation.Loan{
		ID:         d.ID,
		StudentID:  d.StudentID,
		BookID:     d.BookID,
		LoanDate:   d.LoanDate,
		DueDate:    d.DueDate,
		ReturnDate: d.ReturnDate,
		Status:     d.Status,
		WasLate:    d.WasLate,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}

func (r *LoanRepository) FindByID(ctx context.Context, id string) (*circulation.Loan, error) {
	var doc loanDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, circulation.ErrLoanNotFound
	}
	if err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

func (r *LoanRepository) CountActiveByStudent(ctx context.Context, studentID string) (int, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{
		"student_id": studentID,
		"status":     circulation.StatusActive,
	})
	return int(count), err
}

// Search builds a filtered query for HU-07/HU-08 (loan history, overdue loans).
func (r *LoanRepository) Search(ctx context.Context, status string, overdueOnly bool, studentID, bookID string, page, limit int) ([]*circulation.Loan, int, error) {
	filter := bson.M{}
	if status != "" {
		filter["status"] = status
	}
	if overdueOnly {
		filter["status"] = circulation.StatusActive
		filter["due_date"] = bson.M{"$lt": time.Now().UTC()}
	}
	if studentID != "" {
		filter["student_id"] = studentID
	}
	if bookID != "" {
		filter["book_id"] = bookID
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	opts := options.Find().
		SetSort(bson.D{{Key: "loan_date", Value: -1}}).
		SetSkip(int64(offset)).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var loans []*circulation.Loan
	for cursor.Next(ctx) {
		var doc loanDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, 0, err
		}
		loans = append(loans, doc.toDomain())
	}
	if err := cursor.Err(); err != nil {
		return nil, 0, err
	}

	return loans, int(total), nil
}

func (r *LoanRepository) Save(ctx context.Context, l *circulation.Loan) error {
	doc := toDocument(l)
	opts := options.Replace().SetUpsert(true)
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": doc.ID}, doc, opts)
	return err
}

package persistence

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const idempotencyKeysCollection = "idempotency_keys"

// IdempotencyStore implements out.IdempotencyStore against MongoDB's
// idempotency_keys collection (lms-circulation-db). Durable — replaces the
// provisional in-memory adapter now that the collection exists.
type IdempotencyStore struct {
	collection *mongo.Collection
}

func NewIdempotencyStore(db *mongo.Database) *IdempotencyStore {
	return &IdempotencyStore{collection: db.Collection(idempotencyKeysCollection)}
}

type idempotencyKeyDocument struct {
	Key       string    `bson:"_id"`
	LoanID    string    `bson:"loan_id"`
	CreatedAt time.Time `bson:"created_at"`
}

func (s *IdempotencyStore) Get(ctx context.Context, key string) (loanID string, found bool, err error) {
	var doc idempotencyKeyDocument
	err = s.collection.FindOne(ctx, bson.M{"_id": key}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return doc.LoanID, true, nil
}

// Save is an upsert on _id (the key itself), same pattern as LoanRepository.Save
// — RegisterLoan only ever calls this once per key, but ReplaceOne with
// upsert keeps it safe either way.
func (s *IdempotencyStore) Save(ctx context.Context, key, loanID string) error {
	doc := idempotencyKeyDocument{Key: key, LoanID: loanID, CreatedAt: time.Now().UTC()}
	opts := options.Replace().SetUpsert(true)
	_, err := s.collection.ReplaceOne(ctx, bson.M{"_id": key}, doc, opts)
	return err
}

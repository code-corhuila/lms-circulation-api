// Package idempotency implements application/port/out.IdempotencyStore.
//
// Provisional: this is an in-memory map, not a durable collection in
// lms-circulation-db (rules/2-anexos/B-db-mongo.md) — that repo does not have
// one yet (ADR-010-liquibase-for-database-migrations.md in library-docs
// tracks the -db restructuring this depends on). It does not survive a
// restart and does not coordinate across more than one running instance.
// Replace with a MongoDB-backed adapter once that collection exists; the
// port (out.IdempotencyStore) does not change.
package idempotency

import (
	"context"
	"sync"
)

// MemoryStore is a process-local, mutex-guarded IdempotencyStore.
type MemoryStore struct {
	mu    sync.RWMutex
	byKey map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{byKey: map[string]string{}}
}

func (s *MemoryStore) Get(_ context.Context, key string) (loanID string, found bool, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	loanID, found = s.byKey[key]
	return loanID, found, nil
}

func (s *MemoryStore) Save(_ context.Context, key, loanID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byKey[key] = loanID
	return nil
}

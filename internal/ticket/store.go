package ticket

import (
	"errors"
	"sort"
	"sync"
)

var ErrNotFound = errors.New("ticket not found")

// Store defines the persistence contract for tickets. Swap in a
// SQLite/Postgres-backed implementation later without touching
// handler code.
type Store interface {
	Create(t *Ticket) error
	Update(t *Ticket) error
	FindByID(id string) (*Ticket, error)
	ListByUser(userID string) ([]*Ticket, error)
}

// MemoryStore is a simple thread-safe in-memory implementation of Store.
type MemoryStore struct {
	mu   sync.RWMutex
	byID map[string]*Ticket
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{byID: make(map[string]*Ticket)}
}

func (s *MemoryStore) Create(t *Ticket) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[t.ID] = t
	return nil
}

func (s *MemoryStore) Update(t *Ticket) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byID[t.ID]; !ok {
		return ErrNotFound
	}
	s.byID[t.ID] = t
	return nil
}

func (s *MemoryStore) FindByID(id string) (*Ticket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.byID[id]
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

func (s *MemoryStore) ListByUser(userID string) ([]*Ticket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Ticket
	for _, t := range s.byID {
		if t.UserID == userID {
			result = append(result, t)
		}
	}
	// Deterministic ordering (newest first) since map iteration order
	// is randomized in Go.
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

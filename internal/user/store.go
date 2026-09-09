package user

import (
	"errors"
	"sync"
)

var ErrNotFound = errors.New("user not found")
var ErrDuplicateEmail = errors.New("email already registered")

// Store defines the persistence contract for users. The in-memory
// implementation below satisfies it; swap in a SQLite/Postgres-backed
// implementation later without touching handler code.
type Store interface {
	Create(u *User) error
	FindByEmail(email string) (*User, error)
	FindByID(id string) (*User, error)
}

// MemoryStore is a simple thread-safe in-memory implementation of Store.
type MemoryStore struct {
	mu        sync.RWMutex
	byID      map[string]*User
	emailToID map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		byID:      make(map[string]*User),
		emailToID: make(map[string]string),
	}
}

func (s *MemoryStore) Create(u *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.emailToID[u.Email]; exists {
		return ErrDuplicateEmail
	}

	s.byID[u.ID] = u
	s.emailToID[u.Email] = u.ID
	return nil
}

func (s *MemoryStore) FindByEmail(email string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.emailToID[email]
	if !ok {
		return nil, ErrNotFound
	}
	return s.byID[id], nil
}

func (s *MemoryStore) FindByID(id string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.byID[id]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

package ticket

import (
	"errors"
	"sort"
	"sync"
)

var ErrNotFound = errors.New("ticket not found")

type Store interface {
	Create(t *Ticket) error
	Update(t *Ticket) error
	FindByID(id string) (*Ticket, error)
	ListByUser(userID string) ([]*Ticket, error)
}

type MemoryStore struct {
	mu   sync.RWMutex
	byID map[string]*Ticket
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{byID: make(map[string]*Ticket)}
}

// copy returns a shallow copy of a ticket. Handlers get a private copy to
// mutate freely — the only value that ever lives in the store's map is
// the one written back through Update, under the lock. This closes a
// data race where a handler could mutate a ticket's fields (e.g.
// UpdateStatus setting t.Status) at the same moment another goroutine's
// ListByUser/FindByID reads that same shared struct.
func copyTicket(t *Ticket) *Ticket {
	c := *t
	return &c
}

func (s *MemoryStore) Create(t *Ticket) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[t.ID] = copyTicket(t)
	return nil
}

func (s *MemoryStore) Update(t *Ticket) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byID[t.ID]; !ok {
		return ErrNotFound
	}
	s.byID[t.ID] = copyTicket(t)
	return nil
}

func (s *MemoryStore) FindByID(id string) (*Ticket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.byID[id]
	if !ok {
		return nil, ErrNotFound
	}
	return copyTicket(t), nil
}

func (s *MemoryStore) ListByUser(userID string) ([]*Ticket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Ticket
	for _, t := range s.byID {
		if t.UserID == userID {
			result = append(result, copyTicket(t))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

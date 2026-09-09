package ticket

import "time"

type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusClosed     Status = "closed"
)

func (s Status) Valid() bool {
	switch s {
	case StatusOpen, StatusInProgress, StatusClosed:
		return true
	default:
		return false
	}
}

// validTransitions encodes the allowed status flow:
//
//	open -> in_progress -> closed
//
// closed is terminal: it cannot move back to open or in_progress.
var validTransitions = map[Status]Status{
	StatusOpen:       StatusInProgress,
	StatusInProgress: StatusClosed,
}

// CanTransition reports whether moving from `from` to `to` is allowed.
func CanTransition(from, to Status) bool {
	next, ok := validTransitions[from]
	return ok && next == to
}

// Ticket represents a single support ticket owned by a user.
type Ticket struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

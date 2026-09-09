package user

import "time"

// User represents a registered account. PasswordHash is never
// serialized to JSON responses.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

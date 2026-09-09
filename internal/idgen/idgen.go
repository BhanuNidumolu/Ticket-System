// Package idgen generates random, URL-safe unique IDs using only the
// standard library (crypto/rand), avoiding a dependency on
// google/uuid or similar.
package idgen

import (
	"crypto/rand"
	"encoding/hex"
)

// New returns a random 16-byte ID hex-encoded to a 32-character string.
func New() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failing is effectively unrecoverable (broken
		// system entropy source); panic is appropriate here.
		panic("idgen: failed to read random bytes: " + err.Error())
	}
	return hex.EncodeToString(b)
}

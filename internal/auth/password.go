package auth

// This file implements password hashing using only the Go standard
// library (crypto/hmac, crypto/sha256, crypto/rand). It deliberately
// avoids golang.org/x/crypto/bcrypt so the project has zero external
// dependencies and can be built fully offline with `go build`.
//
// The scheme is a straightforward PBKDF2-HMAC-SHA256 implementation:
// a random salt is generated per user, HMAC-SHA256 is applied
// iteratively, and the encoded output stores the iteration count and
// salt alongside the derived hash so verification is self-describing
// (similar in spirit to how bcrypt embeds its cost factor).
//
// If you want to swap this for golang.org/x/crypto/bcrypt later, only
// this file needs to change — HashPassword/VerifyPassword is the
// contract the rest of the app relies on.

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

const (
	pbkdf2Iterations = 100_000
	saltBytes        = 16
	keyBytes         = 32
)

// pbkdf2 derives a key of length keyLen from password+salt using
// HMAC-SHA256 as the pseudorandom function, run for `iterations` rounds.
func pbkdf2(password, salt []byte, iterations, keyLen int) []byte {
	prf := hmac.New(sha256.New, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen

	var derivedKey []byte
	for block := 1; block <= numBlocks; block++ {
		prf.Reset()
		prf.Write(salt)
		prf.Write([]byte{
			byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block),
		})
		u := prf.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)

		for i := 1; i < iterations; i++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		derivedKey = append(derivedKey, t...)
	}
	return derivedKey[:keyLen]
}

// HashPassword returns an encoded hash string of the form:
//
//	pbkdf2-sha256$<iterations>$<salt-hex>$<hash-hex>
//
// This whole string is safe to store in place of the plaintext password.
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	derived := pbkdf2([]byte(password), salt, pbkdf2Iterations, keyBytes)

	return fmt.Sprintf(
		"pbkdf2-sha256$%d$%s$%s",
		pbkdf2Iterations,
		hex.EncodeToString(salt),
		hex.EncodeToString(derived),
	), nil
}

// VerifyPassword checks a plaintext password against a hash produced by
// HashPassword. It returns true only on an exact match, using a
// constant-time comparison to avoid timing side channels.
func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false, fmt.Errorf("unrecognized password hash format")
	}

	iterations, err := strconv.Atoi(parts[1])
	if err != nil {
		return false, fmt.Errorf("invalid iteration count: %w", err)
	}
	salt, err := hex.DecodeString(parts[2])
	if err != nil {
		return false, fmt.Errorf("invalid salt encoding: %w", err)
	}
	expected, err := hex.DecodeString(parts[3])
	if err != nil {
		return false, fmt.Errorf("invalid hash encoding: %w", err)
	}

	actual := pbkdf2([]byte(password), salt, iterations, len(expected))
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

package auth

import (
	"strings"
	"testing"
)

func TestHashPassword_ProducesExpectedFormat(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	parts := strings.Split(hash, "$")
	if len(parts) != 4 {
		t.Fatalf("expected 4 '$'-separated parts, got %d: %q", len(parts), hash)
	}
	if parts[0] != "pbkdf2-sha256" {
		t.Errorf("expected algorithm prefix pbkdf2-sha256, got %q", parts[0])
	}
}

func TestHashPassword_NeverStoresPlaintext(t *testing.T) {
	password := "correct-horse-battery-staple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if strings.Contains(hash, password) {
		t.Errorf("hash appears to contain the raw plaintext password")
	}
}

func TestVerifyPassword_CorrectPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	ok, err := VerifyPassword("correct-horse-battery-staple", hash)
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if !ok {
		t.Error("expected VerifyPassword to succeed with the correct password")
	}
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	ok, err := VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if ok {
		t.Error("expected VerifyPassword to fail with an incorrect password")
	}
}

func TestVerifyPassword_TwoUsersSamePassword_DifferentHashes(t *testing.T) {
	h1, _ := HashPassword("same-password")
	h2, _ := HashPassword("same-password")

	if h1 == h2 {
		t.Error("expected different salts to produce different hashes for the same password")
	}
}

func TestVerifyPassword_MalformedHash(t *testing.T) {
	_, err := VerifyPassword("anything", "not-a-real-hash")
	if err == nil {
		t.Error("expected an error for a malformed stored hash")
	}
}

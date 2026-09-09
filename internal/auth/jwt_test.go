package auth

import (
	"errors"
	"testing"
	"time"
)

func TestGenerateAndParseToken_RoundTrip(t *testing.T) {
	secret := []byte("test-secret")

	token, err := GenerateToken("user-123", secret, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := ParseToken(token, secret)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("claims.UserID = %q, want %q", claims.UserID, "user-123")
	}
}

func TestParseToken_WrongSecretRejected(t *testing.T) {
	token, err := GenerateToken("user-123", []byte("secret-a"), time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	_, err = ParseToken(token, []byte("secret-b"))
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken for wrong secret, got %v", err)
	}
}

func TestParseToken_ExpiredTokenRejected(t *testing.T) {
	secret := []byte("test-secret")

	// ttl of -1 second means it's already expired the instant it's issued.
	token, err := GenerateToken("user-123", secret, -1*time.Second)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	_, err = ParseToken(token, secret)
	if !errors.Is(err, ErrExpiredToken) {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestParseToken_MalformedTokenRejected(t *testing.T) {
	_, err := ParseToken("not.a.validtoken", []byte("test-secret"))
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken for malformed token, got %v", err)
	}
}

func TestParseToken_TamperedPayloadRejected(t *testing.T) {
	secret := []byte("test-secret")
	token, err := GenerateToken("user-123", secret, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Flip the last character of the token to simulate tampering.
	tampered := token[:len(token)-1] + "x"

	_, err = ParseToken(tampered, secret)
	if err == nil {
		t.Error("expected an error for a tampered token, got nil")
	}
}

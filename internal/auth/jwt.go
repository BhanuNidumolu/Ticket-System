package auth

// A minimal HS256 JWT implementation using only the standard library
// (crypto/hmac, crypto/sha256, encoding/json, encoding/base64). This
// avoids an external dependency like golang-jwt/jwt so the module has
// no third-party requirements at all.
//
// It intentionally supports only what this project needs: a single
// "sub" (user id) claim plus an expiry ("exp"). If your requirements
// grow (multiple audiences, refresh tokens, RS256, etc.) swapping in
// golang-jwt/jwt is a drop-in replacement — Generate/ParseToken is the
// contract the rest of the app depends on.

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

type header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// Claims is the JWT payload used across the app.
type Claims struct {
	UserID    string `json:"sub"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

func b64Encode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func b64Decode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// GenerateToken creates a signed HS256 JWT for the given user id, valid
// for ttl from now.
func GenerateToken(userID string, secret []byte, ttl time.Duration) (string, error) {
	now := time.Now()
	h := header{Alg: "HS256", Typ: "JWT"}
	c := Claims{
		UserID:    userID,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(ttl).Unix(),
	}

	headerJSON, err := json.Marshal(h)
	if err != nil {
		return "", fmt.Errorf("marshal header: %w", err)
	}
	claimsJSON, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}

	signingInput := b64Encode(headerJSON) + "." + b64Encode(claimsJSON)
	sig := sign(signingInput, secret)

	return signingInput + "." + b64Encode(sig), nil
}

// ParseToken validates the signature and expiry of a JWT and returns
// its claims.
func ParseToken(token string, secret []byte) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSig := sign(signingInput, secret)

	actualSig, err := b64Decode(parts[2])
	if err != nil {
		return nil, ErrInvalidToken
	}
	if subtle.ConstantTimeCompare(expectedSig, actualSig) != 1 {
		return nil, ErrInvalidToken
	}

	claimsJSON, err := b64Decode(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var c Claims
	if err := json.Unmarshal(claimsJSON, &c); err != nil {
		return nil, ErrInvalidToken
	}

	if time.Now().Unix() > c.ExpiresAt {
		return nil, ErrExpiredToken
	}

	return &c, nil
}

func sign(input string, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(input))
	return mac.Sum(nil)
}

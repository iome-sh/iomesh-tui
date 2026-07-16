package mcp

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

func pkceS256Impl(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// NewPKCEVerifier returns a high-entropy code_verifier (43–128 chars, unreserved).
func NewPKCEVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

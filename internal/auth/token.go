package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const (
	// TokenLength is the length of generated tokens in bytes (before base64 encoding)
	TokenLength = 32
)

// GenerateToken generates a cryptographically secure random token
// Returns the raw token (to be shown to user once) and the hash (to be stored)
func GenerateToken() (token string, hash string, err error) {
	bytes := make([]byte, TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Use URL-safe base64 encoding for the token
	token = base64.URLEncoding.EncodeToString(bytes)

	// Hash the token for storage
	hash = HashToken(token)

	return token, hash, nil
}

// HashToken creates a SHA-256 hash of the token for secure storage
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// ValidateToken checks if a provided token matches the stored hash
func ValidateToken(token, storedHash string) bool {
	computedHash := HashToken(token)
	return subtle.ConstantTimeCompare([]byte(computedHash), []byte(storedHash)) == 1
}

// GenerateAdminSetupToken generates a one-time admin setup token
// This is used during initial setup when no admin account exists
func GenerateAdminSetupToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate setup token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// MaskToken masks a token for display purposes (shows only first and last 4 chars)
func MaskToken(token string) string {
	if len(token) <= 12 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

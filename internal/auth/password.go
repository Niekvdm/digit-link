package auth

import (
	"fmt"
	"os"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultBcryptCost is the default cost factor for bcrypt hashing
	DefaultBcryptCost = 12
	// MinBcryptCost is the minimum allowed bcrypt cost
	MinBcryptCost = 10
	// MaxBcryptCost is the maximum allowed bcrypt cost
	MaxBcryptCost = 16
)

// getBcryptCost returns the bcrypt cost factor from environment or default
func getBcryptCost() int {
	if costStr := os.Getenv("BCRYPT_COST"); costStr != "" {
		if cost, err := strconv.Atoi(costStr); err == nil {
			if cost >= MinBcryptCost && cost <= MaxBcryptCost {
				return cost
			}
		}
	}
	return DefaultBcryptCost
}

// HashPassword creates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", fmt.Errorf("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), getBcryptCost())
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// VerifyPassword checks if the provided password matches the stored hash
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// JWTExpiration is the default expiration time for JWT tokens
	JWTExpiration = 24 * time.Hour
)

// JWTClaims contains the claims for a JWT token
type JWTClaims struct {
	AccountID string `json:"accountId"`
	Username  string `json:"username"`
	IsAdmin   bool   `json:"isAdmin"`
	OrgID     string `json:"orgId,omitempty"` // Set for org accounts
	jwt.RegisteredClaims
}

// jwtSecret holds the cached JWT secret
var jwtSecret []byte

// getJWTSecret returns the JWT signing secret
func getJWTSecret() ([]byte, error) {
	if jwtSecret != nil {
		return jwtSecret, nil
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Check if we're in production mode - fail if so
		env := os.Getenv("ENV")
		if env == "production" || os.Getenv("PRODUCTION") == "true" {
			return nil, fmt.Errorf("JWT_SECRET environment variable must be set in production mode")
		}

		// Generate a random secret if not provided (development only)
		randomBytes := make([]byte, 32)
		if _, err := rand.Read(randomBytes); err != nil {
			return nil, fmt.Errorf("failed to generate JWT secret: %w", err)
		}
		secret = hex.EncodeToString(randomBytes)
		// Log warning using structured logging
		log.Printf("WARNING: JWT_SECRET not set, using auto-generated secret. Sessions will not persist across restarts.")
	}

	jwtSecret = []byte(secret)
	return jwtSecret, nil
}

// GenerateJWT creates a new JWT token for an authenticated user
func GenerateJWT(accountID, username string, isAdmin bool) (string, error) {
	return GenerateJWTWithOrg(accountID, username, isAdmin, "")
}

// GenerateJWTWithOrg creates a new JWT token for an authenticated user with optional org context
func GenerateJWTWithOrg(accountID, username string, isAdmin bool, orgID string) (string, error) {
	secret, err := getJWTSecret()
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := JWTClaims{
		AccountID: accountID,
		Username:  username,
		IsAdmin:   isAdmin,
		OrgID:     orgID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(JWTExpiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "digit-link",
			Subject:   accountID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ValidateJWT validates a JWT token and returns the claims
func ValidateJWT(tokenString string) (*JWTClaims, error) {
	secret, err := getJWTSecret()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// GeneratePendingToken creates a short-lived token for the TOTP verification step
// This token is used between password verification and TOTP verification
func GeneratePendingToken(accountID, username string) (string, error) {
	secret, err := getJWTSecret()
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"accountId": accountID,
		"username":  username,
		"pending":   true, // Marks this as a pending authentication
		"exp":       now.Add(5 * time.Minute).Unix(),
		"iat":       now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ValidatePendingToken validates a pending authentication token
func ValidatePendingToken(tokenString string) (accountID string, username string, err error) {
	secret, err := getJWTSecret()
	if err != nil {
		return "", "", err
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		return "", "", fmt.Errorf("invalid pending token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		pending, _ := claims["pending"].(bool)
		if !pending {
			return "", "", fmt.Errorf("not a pending token")
		}

		accountID, _ = claims["accountId"].(string)
		username, _ = claims["username"].(string)

		if accountID == "" || username == "" {
			return "", "", fmt.Errorf("invalid pending token claims")
		}

		return accountID, username, nil
	}

	return "", "", fmt.Errorf("invalid pending token")
}

package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	// TOTPIssuer is the issuer name shown in authenticator apps
	TOTPIssuer = "digit-link"
)

// TOTPKey contains the generated TOTP secret and provisioning URL
type TOTPKey struct {
	Secret string `json:"secret"`
	URL    string `json:"url"`
}

// GenerateTOTPSecret generates a new TOTP secret for an account
func GenerateTOTPSecret(username string) (*TOTPKey, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      TOTPIssuer,
		AccountName: username,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	return &TOTPKey{
		Secret: key.Secret(),
		URL:    key.URL(),
	}, nil
}

// ValidateTOTP validates a TOTP code against the secret
func ValidateTOTP(secret, code string) bool {
	return totp.Validate(code, secret)
}

// ValidateTOTPWithWindow validates a TOTP code with a time window for clock drift
func ValidateTOTPWithWindow(secret, code string) bool {
	// Allow 1 period before and after for clock drift
	valid, _ := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	return valid
}

// getEncryptionKey derives an AES-256 key from the JWT secret
func getEncryptionKey() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable not set")
	}

	// Derive a 32-byte key using SHA-256
	hash := sha256.Sum256([]byte(secret))
	return hash[:], nil
}

// EncryptTOTPSecret encrypts the TOTP secret for storage
func EncryptTOTPSecret(secret string) (string, error) {
	key, err := getEncryptionKey()
	if err != nil {
		// If no JWT_SECRET, store unencrypted (for development)
		return secret, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(secret), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptTOTPSecret decrypts the stored TOTP secret
func DecryptTOTPSecret(encrypted string) (string, error) {
	key, err := getEncryptionKey()
	if err != nil {
		// If no JWT_SECRET, assume unencrypted
		return encrypted, nil
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		// Might be unencrypted, return as-is
		return encrypted, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		// Too short to be encrypted, return as-is
		return encrypted, nil
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// Decryption failed, might be unencrypted
		return encrypted, nil
	}

	return string(plaintext), nil
}

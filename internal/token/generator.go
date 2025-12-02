package token

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateToken generates a random secure token
func GenerateToken(length int) (string, error) {
	if length <= 0 {
		length = 32
	}

	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	return base64.URLEncoding.EncodeToString(b), nil
}

// GeneratePassword generates a random password
func GeneratePassword() (string, error) {
	// Generate a 24-byte random password (32 characters when base64 encoded)
	return GenerateToken(24)
}

// GenerateUsername generates a random username
func GenerateUsername(prefix string) (string, error) {
	token, err := GenerateToken(8)
	if err != nil {
		return "", err
	}

	if prefix == "" {
		prefix = "user"
	}

	return fmt.Sprintf("%s-%s", prefix, token[:12]), nil
}

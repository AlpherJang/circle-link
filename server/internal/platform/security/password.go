package security

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const passwordIterations = 120000

var ErrInvalidPasswordHash = errors.New("invalid password hash")

// HashPassword is a temporary stdlib-only implementation for local scaffolding.
// Replace with Argon2id before production.
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	sum := derivePasswordHash(password, salt)
	return fmt.Sprintf("sha256$%d$%s$%s", passwordIterations, hex.EncodeToString(salt), hex.EncodeToString(sum)), nil
}

func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "sha256" {
		return false, ErrInvalidPasswordHash
	}

	salt, err := hex.DecodeString(parts[2])
	if err != nil {
		return false, err
	}

	expected, err := hex.DecodeString(parts[3])
	if err != nil {
		return false, err
	}

	sum := derivePasswordHash(password, salt)
	if subtle.ConstantTimeCompare(sum, expected) != 1 {
		return false, nil
	}

	return true, nil
}

func derivePasswordHash(password string, salt []byte) []byte {
	payload := append(append([]byte{}, salt...), []byte(password)...)
	sum := sha256.Sum256(payload)
	output := sum[:]

	for i := 1; i < passwordIterations; i++ {
		nextPayload := append(append([]byte{}, output...), salt...)
		next := sha256.Sum256(nextPayload)
		output = next[:]
	}

	return output
}

package crypto

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashToken returns the hex-encoded SHA-256 hash of a token string.
func HashToken(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

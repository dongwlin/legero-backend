package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2MemoryKiB   uint32 = 64 * 1024
	argon2Iterations  uint32 = 3
	argon2Parallelism uint8  = 2
	argon2SaltLength  uint32 = 16
	argon2KeyLength   uint32 = 32
)

// PasswordHasher hashes and compares passwords using Argon2id.
type PasswordHasher struct{}

// NewPasswordHasher creates a PasswordHasher with the application's fixed
// Argon2id parameters.
func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{}
}

// Hash returns an Argon2id encoded hash of password.
func (h *PasswordHasher) Hash(password string) (string, error) {
	salt := make([]byte, argon2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read password salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argon2Iterations,
		argon2MemoryKiB,
		argon2Parallelism,
		argon2KeyLength,
	)

	saltEncoded := base64.RawStdEncoding.EncodeToString(salt)
	hashEncoded := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2MemoryKiB,
		argon2Iterations,
		argon2Parallelism,
		saltEncoded,
		hashEncoded,
	), nil
}

// Compare checks whether password matches the encoded Argon2id hash.
func (h *PasswordHasher) Compare(password, encodedHash string) (bool, error) {
	params, salt, expectedHash, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	computed := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.MemoryKiB,
		params.Parallelism,
		params.KeyLength,
	)

	if subtle.ConstantTimeCompare(expectedHash, computed) == 1 {
		return true, nil
	}
	return false, nil
}

type argon2HashParams struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	KeyLength   uint32
}

func decodeHash(encodedHash string) (argon2HashParams, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return argon2HashParams{}, nil, nil, fmt.Errorf("invalid password hash format")
	}
	if parts[1] != "argon2id" {
		return argon2HashParams{}, nil, nil, fmt.Errorf("unsupported password hash algorithm")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return argon2HashParams{}, nil, nil, fmt.Errorf("unsupported password hash version")
	}

	var memory uint32
	var iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return argon2HashParams{}, nil, nil, fmt.Errorf("invalid password hash parameters")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argon2HashParams{}, nil, nil, fmt.Errorf("decode password salt: %w", err)
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argon2HashParams{}, nil, nil, fmt.Errorf("decode password hash: %w", err)
	}

	return argon2HashParams{
		MemoryKiB:   memory,
		Iterations:  iterations,
		Parallelism: parallelism,
		KeyLength:   uint32(len(hash)),
	}, salt, hash, nil
}

package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"

	"github.com/dongwlin/legero-backend/internal/infra/config"
)

type PasswordHasher struct {
	config config.Argon2Config
}

func NewPasswordHasher(cfg config.Argon2Config) *PasswordHasher {
	return &PasswordHasher{config: cfg}
}

func (h *PasswordHasher) Hash(password string) (string, error) {
	salt := make([]byte, h.config.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read password salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		h.config.Iterations,
		h.config.MemoryKiB,
		h.config.Parallelism,
		h.config.KeyLength,
	)

	saltEncoded := base64.RawStdEncoding.EncodeToString(salt)
	hashEncoded := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		h.config.MemoryKiB,
		h.config.Iterations,
		h.config.Parallelism,
		saltEncoded,
		hashEncoded,
	), nil
}

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

func decodeHash(encodedHash string) (config.Argon2Config, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return config.Argon2Config{}, nil, nil, fmt.Errorf("invalid password hash format")
	}
	if parts[1] != "argon2id" {
		return config.Argon2Config{}, nil, nil, fmt.Errorf("unsupported password hash algorithm")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return config.Argon2Config{}, nil, nil, fmt.Errorf("unsupported password hash version")
	}

	var memory uint32
	var iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return config.Argon2Config{}, nil, nil, fmt.Errorf("invalid password hash parameters")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return config.Argon2Config{}, nil, nil, fmt.Errorf("decode password salt: %w", err)
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return config.Argon2Config{}, nil, nil, fmt.Errorf("decode password hash: %w", err)
	}

	return config.Argon2Config{
		MemoryKiB:   memory,
		Iterations:  iterations,
		Parallelism: parallelism,
		SaltLength:  uint32(len(salt)),
		KeyLength:   uint32(len(hash)),
	}, salt, hash, nil
}

func ParseArgon2HashCost(encodedHash string) (string, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) < 4 {
		return "", fmt.Errorf("invalid password hash format")
	}
	return parts[3], nil
}

func MustHashForTests(password string) string {
	hasher := NewPasswordHasher(config.Argon2Config{
		MemoryKiB:   64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	})
	hash, err := hasher.Hash(password)
	if err != nil {
		panic(err)
	}
	return hash
}

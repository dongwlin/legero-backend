package crypto

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/crypto/argon2"
)

func TestPasswordHasherHashAndCompare(t *testing.T) {
	hasher := NewPasswordHasher()

	hash, err := hasher.Hash("secret")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Fatalf("Hash() parameters = %q, want fixed Argon2id defaults", hash)
	}

	matched, err := hasher.Compare("secret", hash)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if !matched {
		t.Fatal("expected password comparison to succeed")
	}

	matched, err = hasher.Compare("wrong", hash)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if matched {
		t.Fatal("expected password comparison to fail for wrong password")
	}
}

func TestPasswordHasherCompareSupportsNonDefaultParameters(t *testing.T) {
	const (
		password    = "legacy-secret"
		memoryKiB   = 8 * 1024
		iterations  = 1
		parallelism = 1
		keyLength   = 32
	)
	salt := []byte("0123456789abcdef")
	hash := argon2.IDKey([]byte(password), salt, iterations, memoryKiB, parallelism, keyLength)
	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		memoryKiB,
		iterations,
		parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	hasher := NewPasswordHasher()
	matched, err := hasher.Compare(password, encodedHash)
	if err != nil {
		t.Fatalf("Compare() legacy hash error = %v", err)
	}
	if !matched {
		t.Fatal("expected legacy Argon2id parameters to verify the correct password")
	}

	matched, err = hasher.Compare("wrong-password", encodedHash)
	if err != nil {
		t.Fatalf("Compare() legacy hash with wrong password error = %v", err)
	}
	if matched {
		t.Fatal("expected legacy Argon2id parameters to reject the wrong password")
	}
}

package crypto

import (
	"testing"

	"github.com/dongwlin/legero-backend/internal/infra/config"
)

func TestPasswordHasherHashAndCompare(t *testing.T) {
	hasher := NewPasswordHasher(config.Argon2Config{
		MemoryKiB:   8 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	})

	hash, err := hasher.Hash("secret")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
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

package auth

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

func TestNormalizePhone(t *testing.T) {
	got := NormalizePhone("+86 138-0013-8000")
	if got != "13800138000" {
		t.Fatalf("NormalizePhone() = %q, want %q", got, "13800138000")
	}
}

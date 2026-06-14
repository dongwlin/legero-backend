package crypto

import "github.com/dongwlin/legero-backend/internal/infra/config"

// MustHashForTests returns an Argon2id hash of password. It panics on error
// and is intended for use in tests only.
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

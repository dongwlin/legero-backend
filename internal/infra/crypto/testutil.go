package crypto

// MustHashForTests returns an Argon2id hash of password. It panics on error
// and is intended for use in tests only.
func MustHashForTests(password string) string {
	hasher := NewPasswordHasher()
	hash, err := hasher.Hash(password)
	if err != nil {
		panic(err)
	}
	return hash
}

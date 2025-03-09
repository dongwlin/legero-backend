package token

import (
	"crypto/rand"
	"math/big"
)

func Generate(byteLength int) (string, error) {
	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	num := new(big.Int).SetBytes(bytes)
	return num.Text(62), nil
}

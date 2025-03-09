package token

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func isValidBase62String(s string) bool {
	validChars := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	for _, c := range s {
		if !strings.ContainsRune(validChars, c) {
			return false
		}
	}
	return true
}

func TestGenerate(t *testing.T) {
	token, err := Generate(32)
	require.NoError(t, err)
	require.True(t, isValidBase62String(token))
}

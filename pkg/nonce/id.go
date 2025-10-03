package nonce

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"strings"
)

type Nonce string

const NonceLength = 22

const BASE58_ALPHABET = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// Deterministic method to generate an Id
func NewFromSeed(seed []byte) Nonce {
	hash := sha256.Sum256(seed)
	idBytes := hash[:NonceLength]
	idStr := ""
	for _, b := range idBytes {
		idStr += string(BASE58_ALPHABET[int(b)%len(BASE58_ALPHABET)])
	}
	return Nonce(idStr)
}

// Generates a random Id
func Random() Nonce {
	// 32 bytes makes a 22-character string
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return NewFromSeed(b)
}

// To replace uuid.NewString()
func NewString() string {
	return string(Random())
}

func Validate(str string) error {
	if len(str) != NonceLength {
		return fmt.Errorf("treasury-id must be 22 bytes")
	}
	for _, c := range str {
		if !strings.Contains(BASE58_ALPHABET, string(c)) {
			return fmt.Errorf("invalid treasury-id char: %s", string(c))
		}
	}
	return nil
}

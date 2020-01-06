package resourcecreator

import (
	"crypto/rand"
)

// Generate a cryptographically secure random key of N length.
func Keygen(length int) ([]byte, error) {
	buf := make([]byte, length)
	_, err := rand.Read(buf)
	return buf, err
}

package util

import (
	"crypto/rand"
	mathrand "math/rand"
	"time"
)

// Generate a cryptographically secure random key of N length.
func Keygen(length int) ([]byte, error) {
	buf := make([]byte, length)
	_, err := rand.Read(buf)
	return buf, err
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789"

func init() {
	mathrand.Seed(time.Now().UnixNano())
}

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[mathrand.Intn(len(letterBytes))]
	}
	return string(b)
}

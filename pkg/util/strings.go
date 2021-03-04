package util

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/nais/liberator/pkg/namegen"
)

const strTrimMiddleTruncate = "---[truncated]---"
const strTrimRightTruncate = "..."

func StrTrimRight(s string, maxlen int) string {
	l := len(strTrimRightTruncate)
	if maxlen > l {
		return s[:maxlen-l] + strTrimRightTruncate
	}
	return s[:maxlen]
}

func StrTrimMiddle(s string, maxlen int) string {
	if len(s) <= maxlen {
		return s
	}
	newlen := maxlen - len(strTrimMiddleTruncate)
	if newlen < len(strTrimMiddleTruncate) {
		return StrTrimRight(s, maxlen)
	}
	partlen := int(math.Floor(float64(newlen)) / 2)
	return s[:partlen] + strTrimMiddleTruncate + s[len(s)-partlen:]
}

func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func GenerateSecretName(prefix, basename string, maxlen int) (string, error) {
	secretName, err := namegen.ShortName(fmt.Sprintf("%s-%s", prefix, basename), maxlen)
	if err != nil {
		return "", fmt.Errorf("unable to generate '%s' secret name: %s", prefix, err)
	}
	return secretName, err
}

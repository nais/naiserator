package util

import (
	"encoding/base64"
	"fmt"

	"github.com/nais/liberator/pkg/keygen"
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
	partlen := newlen / 2
	return s[:partlen] + strTrimMiddleTruncate + s[len(s)-partlen:]
}

func GeneratePassword() (string, error) {
	key, err := keygen.Keygen(32)
	if err != nil {
		return "", fmt.Errorf("NAISERATOR-8230: unable to generate secret for sql user: %s", err)
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(key), nil
}

package util

import (
	"math"
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

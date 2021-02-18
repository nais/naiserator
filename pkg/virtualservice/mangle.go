package virtualservice

import (
	"regexp"

	"github.com/nais/liberator/pkg/stringutil"
)

var mangleRegex = regexp.MustCompile("[^A-Za-z0-9-]")

func MangleName(host string) string {
	const maxlen = 63
	const replacement = "-"
	mangled := stringutil.UniqueWithHash(host, maxlen)
	return mangleRegex.ReplaceAllString(mangled, replacement)
}

package ingress

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRegexPath(t *testing.T) {
	tests := map[string]bool{
		"/":                 false,
		"/sok":              false,
		"/sok/api/internal": false,
		"/foo.bar":          false, // literal dot is not enough to be a regex
		"/foo-bar_baz":      false,
		`/sok($|\/.+)`:      true,
		`/sok($|\/.+)(/.*)`: true,
		"/baz/(.*)":         true,
		"/api/(.+)/foo":     true,
		"/foo.+":            true,
		"/foo*":             true,
		"/foo$":             true,
		"/foo^bar":          true,
		"/a|b":              true,
		"/foo[0-9]":         true,
	}

	for path, want := range tests {
		assert.Equalf(t, want, isRegexPath(path), "isRegexPath(%q)", path)
	}
}

func TestStripRegexPath(t *testing.T) {
	tests := map[string]string{
		`/sok($|\/.+)`:      "/sok",
		`/sok($|\/.+)(/.*)`: "/sok",
		"/baz/(.*)":         "/baz",
		"/api/(.+)/foo":     "/api",
		"/foo.+":            "/foo",
		"/foo$":             "/foo",
		"/(.*)":             "/",
		"/sok":              "/sok", // no regex, unchanged
	}

	for path, want := range tests {
		assert.Equalf(t, want, stripRegexPath(path), "stripRegexPath(%q)", path)
	}
}

func TestHasCaptureGroupAnnotation(t *testing.T) {
	assert.False(t, hasCaptureGroupAnnotation(nil))
	assert.False(t, hasCaptureGroupAnnotation(map[string]string{}))
	assert.False(t, hasCaptureGroupAnnotation(map[string]string{
		"nginx.ingress.kubernetes.io/rewrite-target": "/",
	}))
	assert.True(t, hasCaptureGroupAnnotation(map[string]string{
		"nginx.ingress.kubernetes.io/rewrite-target": "/$1",
	}))
	assert.True(t, hasCaptureGroupAnnotation(map[string]string{
		"nginx.ingress.kubernetes.io/rewrite-target": "https://other.nav.no/foo/$1$2",
	}))
}

func TestExactPairLiteral(t *testing.T) {
	type result struct {
		literal string
		ok      bool
	}
	tests := map[string]result{
		`/dagpenger/historikk($|\/$)`:     {"/dagpenger/historikk", true},
		`/dagpenger/en/kalkulator($|\/$)`: {"/dagpenger/en/kalkulator", true},
		`/foo($|/$)`:                      {"/foo", true}, // unescaped slash variant
		// prefix-style regexes are not exact pairs
		`/dagpenger/api/history($|\/.*)`: {"", false},
		`/sok($|\/.+)`:                   {"", false},
		`/baz/(.*)`:                      {"", false},
		// not a regex / no suffix
		"/foo": {"", false},
	}

	for path, want := range tests {
		literal, ok := exactPairLiteral(path)
		assert.Equalf(t, want.ok, ok, "exactPairLiteral(%q) ok", path)
		assert.Equalf(t, want.literal, literal, "exactPairLiteral(%q) literal", path)
	}
}

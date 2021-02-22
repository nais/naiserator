package util_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/nais/naiserator/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestValidateUrl(t *testing.T) {
	validURLs := []string{
		"https://foo.bar",
		"https://foo.bar/",
		"https://nais/device",
		"https://na-is/device",
	}

	invalidURLs := []string{
		"http://valid.tld/",          // lacks https
		"https://big-O.tld/notation", // uppercase
		"https://-test.tld/",         // starts with dash
	}

	for _, s := range validURLs {
		u, err := url.Parse(s)
		if err != nil {
			panic(fmt.Errorf("invalid test: unparseable URL '%s'", s))
		}
		assert.NoError(t, util.ValidateUrl(u))
	}

	for _, s := range invalidURLs {
		u, err := url.Parse(s)
		if err != nil {
			panic(fmt.Errorf("invalid test: unparseable URL '%s'", s))
		}
		assert.Error(t, util.ValidateUrl(u), "expected URL '%s' to fail validation", s)
	}
}

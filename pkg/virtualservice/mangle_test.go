package virtualservice_test

import (
	"testing"

	"github.com/nais/naiserator/pkg/virtualservice"
	"github.com/stretchr/testify/assert"
)

func TestMangleName(t *testing.T) {
	tests := map[string]string{
		"www.nav.no":     "www-nav-no-96921c5e",
		"foo.bar.nav.no": "foo-bar-nav-no-21aeee80",
		"foo-bar.nav.no": "foo-bar-nav-no-b84c8881",
		"very-long-name-that-is-significantly-longer-than-sixtythree-characters.foobar.example.com": "very-long-name-that-is-significantly-longer-than-sixty-ea4a941e",
	}
	for in, expected := range tests {
		out := virtualservice.MangleName(in)
		assert.Equal(t, expected, out)
		assert.True(t, len(out) <= 63)
	}
}

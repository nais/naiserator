package ingress

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigrateRewriteTarget(t *testing.T) {
	tests := []struct {
		name             string
		nginxAnnotations map[string]string
		expected         map[string]string
	}{
		{
			name:             "no rewrite-target annotation - no-op",
			nginxAnnotations: map[string]string{},
			expected:         map[string]string{},
		},
		{
			name: "relative path, no capture groups",
			nginxAnnotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/newpath",
			},
			expected: map[string]string{
				"haproxy.org/path-rewrite": "/newpath",
			},
		},
		{
			name: "relative path with nginx capture groups",
			nginxAnnotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/$1/rest",
			},
			expected: map[string]string{
				"haproxy.org/path-rewrite": `/\1/rest`,
			},
		},
		{
			name: "absolute URL, host only (no path)",
			nginxAnnotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "https://example.com",
			},
			expected: map[string]string{
				"haproxy.org/request-redirect":      "example.com",
				"haproxy.org/request-redirect-code": "302",
			},
		},
		{
			name: "absolute URL, host only (root path /)",
			nginxAnnotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "https://example.com/",
			},
			expected: map[string]string{
				"haproxy.org/request-redirect":      "example.com",
				"haproxy.org/request-redirect-code": "302",
			},
		},
		{
			name: "absolute URL with non-root path - uses snippet",
			nginxAnnotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "https://example.com/some/path",
			},
			expected: map[string]string{
				"haproxy.org/backend-config-snippet": "http-request redirect location https://example.com/some/path code 302",
			},
		},
		{
			name: "absolute URL with capture groups - uses snippet",
			nginxAnnotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "https://example.com/$1",
			},
			expected: map[string]string{
				"haproxy.org/backend-config-snippet": `http-request redirect location https://example.com/\1 code 302`,
			},
		},
		{
			name: "absolute URL with several capture groups - uses snippet",
			nginxAnnotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "https://example.com/$1$2",
			},
			expected: map[string]string{
				"haproxy.org/backend-config-snippet": `http-request redirect location https://example.com/\1\2 code 302`,
			},
		},
		{
			name: "absolute URL with arg variable - uses snippet",
			nginxAnnotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "https://example.com/?q=$arg_search",
			},
			expected: map[string]string{
				"haproxy.org/backend-config-snippet": "http-request redirect location https://example.com/?q=%[url_param(search)] code 302",
			},
		},
		{
			name: "absolute URL with capture groups - uses snippet",
			nginxAnnotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "https://syk-dig.intern.dev.nav.no/oppgave/$arg_oppgaveid?",
			},
			expected: map[string]string{
				"haproxy.org/backend-config-snippet": `http-request redirect location https://syk-dig.intern.dev.nav.no/oppgave/%[url_param(oppgaveid)]? code 302`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := map[string]string{}
			migrateRewriteTarget(annotations, tt.nginxAnnotations)
			assert.Equal(t, tt.expected, annotations)
		})
	}
}

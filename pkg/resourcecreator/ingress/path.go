package ingress

import (
	"regexp"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"
)

var (
	// regexPathDetect reports whether an ingress path looks like a regular
	// expression. '.' is intentionally excluded so that literal paths such as "/foo.bar" are not misdetected.
	regexPathDetect = regexp.MustCompile(`[()\[\]{}|^$*+?\\]`)
	// regexPathStrip locates the first regex metacharacter (here '.' is included)
	regexPathStrip = regexp.MustCompile(`[()\[\]{}|^$*+?\\.]`)
	// exactPairSuffix matches the "($|\/$)" / "($|/$)" trailing group, which means
	// "this exact path, optionally with a single trailing slash, and nothing more".
	exactPairSuffix = regexp.MustCompile(`\(\$\|\\?/\$\)$`)
)

// isRegexPath reports whether the ingress path is a regular expression rather
// than a literal path.
func isRegexPath(path string) bool {
	return regexPathDetect.MatchString(path)
}

// stripRegexPath reduces a regex ingress path to its literal prefix so it can be
// used with pathType: Prefix on HAProxy.
func stripRegexPath(path string) string {
	if loc := regexPathStrip.FindStringIndex(path); loc != nil {
		path = path[:loc[0]]
	}
	path = strings.TrimRight(path, "/")
	if path == "" {
		return "/"
	}
	return path
}

// exactPairLiteral reports whether the regex path is of the form
// "<literal>($|\/$)", matching only the literal path and the literal path with a single trailing slash
func exactPairLiteral(path string) (string, bool) {
	loc := exactPairSuffix.FindStringIndex(path)
	if loc == nil {
		return "", false
	}
	literal := strings.TrimRight(path[:loc[0]], "/")
	if literal == "" || isRegexPath(literal) {
		return "", false
	}
	return literal, true
}

// hasCaptureGroupAnnotation reports whether the workload has an nginx annotation
// that depends on regex capture groups from the ingress path (a rewrite-target
// referencing $1, $2, ...). In that case the path regex carries meaning beyond
// matching and must not be stripped.
func hasCaptureGroupAnnotation(annotations map[string]string) bool {
	rewriteTarget := annotations["nginx.ingress.kubernetes.io/rewrite-target"]
	return nginxCaptureGroupRegex.MatchString(rewriteTarget)
}

// haproxyIngressPaths translates an ingress path for the HAProxy ingress
// controller, which uses pathType Prefix/Exact and does not support regular
// expressions.
func haproxyIngressPaths(path string, annotations map[string]string, backend networkingv1.IngressBackend) []networkingv1.HTTPIngressPath {
	if !isRegexPath(path) || hasCaptureGroupAnnotation(annotations) {
		return []networkingv1.HTTPIngressPath{ingressPath(path, networkingv1.PathTypePrefix, backend)}
	}

	if literal, ok := exactPairLiteral(path); ok {
		return []networkingv1.HTTPIngressPath{
			ingressPath(literal, networkingv1.PathTypeExact, backend),
			ingressPath(literal+"/", networkingv1.PathTypeExact, backend),
		}
	}

	return []networkingv1.HTTPIngressPath{ingressPath(stripRegexPath(path), networkingv1.PathTypePrefix, backend)}
}

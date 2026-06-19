package ingress

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const haproxyBackendConfigAnnotation = "haproxy.org/backend-config-snippet"

var (
	nginxCaptureGroupRegex = regexp.MustCompile(`\$(\d+)`)
	nginxArgVariableRegex  = regexp.MustCompile(`\$arg_(\w+)`)
)

func copyHAProxyAnnotations(dst, src map[string]string) {
	for k, v := range src {
		if !strings.HasPrefix(k, "haproxy.org/") {
			continue
		}

		if k == haproxyBackendConfigAnnotation {
			if existing := dst[k]; existing != "" {
				dst[k] = fmt.Sprintf("%s\n\n%s", v, existing)
				continue
			}
		}
		dst[k] = v
	}
}

func copyNginxAnnotations(dst, src map[string]string) {
	for k, v := range src {
		if strings.HasPrefix(k, "nginx.ingress.kubernetes.io/") {
			dst[k] = v
		}
	}
}

func migrateNginxAnnotationsToHAProxyAnnotations(haProxy, nginx map[string]string) {
	nginxAnnotations := map[string]string{}
	copyNginxAnnotations(nginxAnnotations, nginx)

	// Mapping from nginx annotation short key to HAProxy equivalent.
	haProxyAnnotations := map[string]string{
		"keepalive-timeout":     "timeout-http-keep-alive",
		"proxy-connect-timeout": "timeout-connect",
		"proxy-read-timeout":    "timeout-server",
		"proxy-send-timeout":    "timeout-client",
		"upstream-vhost":        "set-host",
	}

	for key, value := range nginxAnnotations {
		nginxKey, _ := strings.CutPrefix(key, "nginx.ingress.kubernetes.io/")
		haProxyKey, performMapping := haProxyAnnotations[nginxKey]
		if !performMapping || haProxyKey == "" {
			continue
		}

		haProxy["haproxy.org/"+haProxyKey] = value
		if strings.HasSuffix(nginxKey, "-timeout") && strings.HasPrefix(haProxyKey, "timeout-") {
			haProxy["haproxy.org/"+haProxyKey] = value + "s"
		}
	}

	migrateLimitRpm(haProxy, nginxAnnotations)
	migrateRewriteTarget(haProxy, nginxAnnotations)
	migrateWhitelistSourceRange(haProxy, nginxAnnotations)
}

func migrateLimitRpm(annotations, nginxAnnotations map[string]string) {
	limitRpm := nginxAnnotations["nginx.ingress.kubernetes.io/limit-rpm"]
	if limitRpm == "" {
		return
	}

	annotations["haproxy.org/rate-limit-period"] = "1m"
	annotations["haproxy.org/rate-limit-requests"] = fmt.Sprint(limitRpm)
	annotations["haproxy.org/rate-limit-status-code"] = "429"
}

// migrateRewriteTarget translates an nginx rewrite-target annotation to equivalent HAProxy annotation(s)
func migrateRewriteTarget(annotations, nginxAnnotations map[string]string) {
	rewriteTarget := nginxAnnotations["nginx.ingress.kubernetes.io/rewrite-target"]
	if rewriteTarget == "" {
		return
	}

	isAbsolute := strings.HasPrefix(rewriteTarget, "http://") || strings.HasPrefix(rewriteTarget, "https://")

	if !isAbsolute {
		annotations["haproxy.org/path-rewrite"] = nginxToHAProxyCaptureGroups(rewriteTarget)
		return
	}

	// HAProxy's request-redirect only supports host substitution and preserves original URI,
	// so path-changing redirects must use backend-config-snippet.
	u, err := url.Parse(rewriteTarget)
	if err != nil {
		migrateRewriteTargetAsSnippet(annotations, rewriteTarget)
		return
	}

	hostOnly := u.Path == "" || u.Path == "/"
	hasCaptures := nginxCaptureGroupRegex.MatchString(rewriteTarget)
	hasArgs := nginxArgVariableRegex.MatchString(rewriteTarget)

	if hostOnly && !hasCaptures && !hasArgs {
		annotations["haproxy.org/request-redirect"] = u.Host
		annotations["haproxy.org/request-redirect-code"] = "302"
		return
	}

	migrateRewriteTargetAsSnippet(annotations, rewriteTarget)
}

func migrateWhitelistSourceRange(annotations, nginxAnnotations map[string]string) {
	whitelistSourceRange := nginxAnnotations["nginx.ingress.kubernetes.io/whitelist-source-range"]
	if whitelistSourceRange == "" {
		return
	}

	ranges := []string{}
	for sourceRange := range strings.SplitSeq(whitelistSourceRange, ",") {
		sourceRange = strings.TrimSpace(sourceRange)
		if sourceRange != "" {
			ranges = append(ranges, sourceRange)
		}
	}
	if len(ranges) == 0 {
		return
	}

	snippet := fmt.Sprintf(
		"acl allowed_src req.hdr_ip(X-Forwarded-For) %s\nhttp-request deny deny_status 403 if !allowed_src",
		strings.Join(ranges, " "),
	)
	delimiter := "# Added by naiserator due to the `nginx.ingress.kubernetes.io/whitelist-source-range` annotation:"
	snippet = fmt.Sprintf("###\n%s\n%s\n###", delimiter, snippet)

	if existing := annotations[haproxyBackendConfigAnnotation]; existing != "" {
		annotations[haproxyBackendConfigAnnotation] = fmt.Sprintf("%s\n\n%s", existing, snippet)
		return
	}
	annotations[haproxyBackendConfigAnnotation] = snippet
}

// nginxToHAProxyCaptureGroups converts $1, $2 to \1, \2
func nginxToHAProxyCaptureGroups(s string) string {
	return nginxCaptureGroupRegex.ReplaceAllString(s, `\$1`)
}

// migrateRewriteTargetAsSnippet uses backend-config-snippet for complex redirects
func migrateRewriteTargetAsSnippet(annotations map[string]string, rewriteTarget string) {
	haproxyTarget := nginxToHAProxyCaptureGroups(rewriteTarget)
	haproxyTarget = nginxArgVariableRegex.ReplaceAllString(haproxyTarget, `%[url_param($1)]`)
	snippet := fmt.Sprintf("http-request redirect location %s code 302", haproxyTarget)
	delimiter := "# Added by naiserator due to the `nginx.ingress.kubernetes.io/rewrite-target` annotation:"
	snippet = fmt.Sprintf("###\n%s\n%s\n###", delimiter, snippet)

	if existing := annotations[haproxyBackendConfigAnnotation]; existing != "" {
		annotations[haproxyBackendConfigAnnotation] = fmt.Sprintf("%s\n\n%s", existing, snippet)
		return
	}
	annotations[haproxyBackendConfigAnnotation] = snippet
}

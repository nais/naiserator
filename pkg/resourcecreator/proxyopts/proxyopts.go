package proxyopts

import (
	"fmt"
	"strings"

	"github.com/nais/naiserator/pkg/proxyopts"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
)

// All pods will have web proxy settings injected as environment variables. This is
// useful for automatic proxy configuration so that apps don't need to be aware
// of infrastructure quirks. The web proxy should be set up as an external service.
//
// Proxy settings on Linux is in a messy state. Some applications and libraries
// read the upper-case variables, while some read the lower-case versions.
// We handle this by setting both versions.
//
// On top of everything, the Java virtual machine does not honor these environment variables.
// Instead, JVM must be started with a specific set of command-line options. These are also
// provided as environment variables, for convenience.
func EnvironmentVariables(options resource.Options) ([]corev1.EnvVar, error) {
	envVars := make([]corev1.EnvVar, 0)

	excludedHosts := options.Proxy.Exclude
	proxyURL := options.Proxy.Address
	noProxy := strings.Join(excludedHosts, ",")

	// Set non-JVM environment variables
	if len(proxyURL) > 0 {
		envVars = appendDualCaseEnvVar(envVars, "HTTP_PROXY", proxyURL)
		envVars = appendDualCaseEnvVar(envVars, "HTTPS_PROXY", proxyURL)
	}
	if len(noProxy) > 0 {
		envVars = appendDualCaseEnvVar(envVars, "NO_PROXY", noProxy)
	}

	// Set environment variables specifically for JVM
	javaOpts, err := proxyopts.JavaProxyOptions(proxyURL, noProxy)
	if err == nil {
		if len(javaOpts) > 0 {
			envVar := corev1.EnvVar{
				Name:  "JAVA_PROXY_OPTIONS",
				Value: javaOpts,
			}
			envVars = append(envVars, envVar)
		}
	} else {
		// A failure state here means that there is something wrong with the syntax
		// of our proxy config. This situation should be made clearly visible.
		return nil, fmt.Errorf("convert webproxy settings to Java format: %w", err)
	}

	return envVars, nil
}

// appendDualCaseEnvVar adds the specified environment variable twice to a slice.
// One with a lowercase key, the other with an uppercase key.
func appendDualCaseEnvVar(envVars []corev1.EnvVar, key, value string) []corev1.EnvVar {
	for _, mkey := range []string{strings.ToUpper(key), strings.ToLower(key)} {
		envVar := corev1.EnvVar{
			Name:  mkey,
			Value: value,
		}
		envVars = append(envVars, envVar)
	}

	return envVars
}

func Create(ast *resource.Ast, resourceOptions resource.Options, webProxyEnabled bool) error {
	if !webProxyEnabled || len(resourceOptions.GoogleProjectId) > 0 {
		return nil
	}

	envs, err := EnvironmentVariables(resourceOptions)
	if err != nil {
		return fmt.Errorf("generate proxy environment variables: %w", err)
	}

	ast.Env = append(ast.Env, envs...)

	return nil
}

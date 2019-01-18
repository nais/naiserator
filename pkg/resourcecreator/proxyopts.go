package resourcecreator

import (
	"fmt"
	"os"
	"strings"

	"github.com/nais/naiserator/pkg/proxyopts"
	corev1 "k8s.io/api/core/v1"
)

const PodHttpProxyEnv = "NAIS_POD_HTTP_PROXY"
const PodNoProxyEnv = "NAIS_POD_NO_PROXY"

func podSpecProxyOpts(podSpec *corev1.PodSpec) (*corev1.PodSpec, error) {
	var err error
	for i := range podSpec.Containers {
		podSpec.Containers[i].Env, err = ProxyEnvironmentVariables(podSpec.Containers[i].Env)
		if err != nil {
			return nil, fmt.Errorf("while injecting proxy options into container: %s", err)
		}
	}
	return podSpec, nil
}

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
func ProxyEnvironmentVariables(envVars []corev1.EnvVar) ([]corev1.EnvVar, error) {
	proxyURL := getEnvDualCase(PodHttpProxyEnv)
	noProxy := getEnvDualCase(PodNoProxyEnv)

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
		return nil, fmt.Errorf("while converting webproxy settings to Java format: %s", err)
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

func getEnvDualCase(name string) string {
	value, found := os.LookupEnv(strings.ToUpper(name))
	if found {
		return value
	}
	return os.Getenv(strings.ToLower(name))
}

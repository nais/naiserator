package test

import (
	"os"
	"strings"
	"testing"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func EnvWrapper(envVars map[string]string, testFunc func(t *testing.T)) func(t *testing.T) {
	return func(t *testing.T) {
		originalEnvVars := make(map[string]string, len(envVars))
		for k, v := range envVars {
			if value, found := os.LookupEnv(k); found {
				originalEnvVars[k] = value
			}
			os.Setenv(k, v)
		}
		testFunc(t)

		for k := range envVars {
			if value, found := originalEnvVars[k]; found {
				os.Setenv(k, value)
			} else {
				os.Unsetenv(k)
			}

		}
	}
}

func NamedResource(resources []runtime.Object, kind string) runtime.Object {
	for _, r := range resources {
		if strings.EqualFold(r.GetObjectKind().GroupVersionKind().Kind, kind) {
			return r
		}
	}
	panic("no matching resource kind found")
}

func EnvValue(envVars []v1.EnvVar, key string) string {
	for _, v := range envVars {
		if v.Name == key {
			return v.Value
		}
	}
	return ""
}

func GetVolumeByName(volumes []v1.Volume, name string) *v1.Volume {
	for _, v := range volumes {
		if v.Name == name {
			return &v
		}
	}

	return nil
}

func GetVolumeMountByName(volumeMounts []v1.VolumeMount, name string) *v1.VolumeMount {
	for _, v := range volumeMounts {
		if v.Name == name {
			return &v
		}
	}

	return nil
}

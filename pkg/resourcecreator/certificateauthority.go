package resourcecreator

import (
	corev1 "k8s.io/api/core/v1"
)

// These constants refer to a ConfigMap that has already been applied to the cluster.
// The source filenames refer to a PEM bundle and a JKS keystore, respectively.
const (
	CA_BUNDLE_CONFIGMAP_NAME      = "ca-bundle-pem"
	CA_BUNDLE_PEM_SOURCE_FILENAME = "ca-bundle.pem"
	CA_BUNDLE_JKS_CONFIGMAP_NAME  = "ca-bundle-jks"
	CA_BUNDLE_JKS_SOURCE_FILENAME = "ca-bundle.jks"
	NAV_TRUSTSTORE_PATH           = "/etc/ssl/certs/java/cacerts"
	NAV_TRUSTSTORE_PASSWORD       = "changeme" // The contents in this file is not secret
)

// The following list was copied from https://golang.org/src/crypto/x509/root_linux.go.
// CA injection should work out of the box. Implementations differ across systems, so
// by mounting the certificates in these places, we increase the chances of successful auto-configuration.
var certFiles = []string{
	"/etc/ssl/certs/ca-certificates.crt",                // Debian/Ubuntu/Gentoo etc.
	"/etc/pki/tls/certs/ca-bundle.crt",                  // Fedora/RHEL 6
	"/etc/ssl/ca-bundle.pem",                            // OpenSUSE
	"/etc/pki/tls/cacert.pem",                           // OpenELEC
	"/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem", // CentOS/RHEL 7
}

// Mount certificate authority files in all locations expected by different kinds of systems.
func certificateAuthorityVolumeMounts() []corev1.VolumeMount {
	vm := make([]corev1.VolumeMount, 0)

	vm = append(vm, corev1.VolumeMount{
		Name:      CA_BUNDLE_JKS_CONFIGMAP_NAME,
		MountPath: NAV_TRUSTSTORE_PATH,
		SubPath:   CA_BUNDLE_JKS_SOURCE_FILENAME,
	})

	for _, path := range certFiles {
		vm = append(vm, corev1.VolumeMount{
			Name:      CA_BUNDLE_CONFIGMAP_NAME,
			MountPath: path,
			SubPath:   CA_BUNDLE_PEM_SOURCE_FILENAME,
		})
	}

	return vm
}

// Configures a Volume to mount files from the CA bundle ConfigMap.
func certificateAuthorityVolume(configMapName string) corev1.Volume {
	return corev1.Volume{
		Name: configMapName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMapName,
				},
			},
		},
	}
}

// Insert the CA configuration into a PodSpec.
func caBundle(podSpec *corev1.PodSpec) *corev1.PodSpec {
	envs := []corev1.EnvVar{
		{
			Name:  "NAV_TRUSTSTORE_PATH",
			Value: NAV_TRUSTSTORE_PATH,
		},
		{
			Name:  "NAV_TRUSTSTORE_PASSWORD",
			Value: NAV_TRUSTSTORE_PASSWORD,
		},
	}

	mainContainer := &podSpec.Containers[0]
	mainContainer.Env = append(mainContainer.Env, envs...)
	mainContainer.VolumeMounts = append(mainContainer.VolumeMounts, certificateAuthorityVolumeMounts()...)

	podSpec.Volumes = append(podSpec.Volumes, certificateAuthorityVolume(CA_BUNDLE_JKS_CONFIGMAP_NAME), certificateAuthorityVolume(CA_BUNDLE_CONFIGMAP_NAME))

	return podSpec
}

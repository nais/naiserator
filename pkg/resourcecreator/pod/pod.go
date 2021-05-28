package pod

import (
	"strconv"
	"strings"

	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	naisAppNameEnv     = "NAIS_APP_NAME"
	naisNamespaceEnv   = "NAIS_NAMESPACE"
	naisAppImageEnv    = "NAIS_APP_IMAGE"
	naisClusterNameEnv = "NAIS_CLUSTER_NAME"
	naisClientId       = "NAIS_CLIENT_ID"
)

func CreateSpec(objectMeta metav1.ObjectMeta, resourceOptions resource.Options, image, preStopHookPath string, appPort int, naisResources nais_io_v1alpha1.ResourceRequirements, livenessProbe, readinessProve, startupProbe *nais_io_v1alpha1.Probe, naisFilesFrom []nais_io_v1alpha1.FilesFrom, naisEnvFrom []nais_io_v1alpha1.EnvFrom, naisEnvVars []nais_io_v1alpha1.EnvVar) (*corev1.PodSpec, error) {
	var err error

	podSpec := podSpecBase(objectMeta, resourceOptions, image, preStopHookPath, appPort, naisResources, livenessProbe, readinessProve, startupProbe, naisEnvVars)

	if len(resourceOptions.HostAliases) > 0 {
		podSpec.HostAliases = hostAliases(resourceOptions)
	}

	podSpec = filesFrom(podSpec, resourceOptions.NativeSecrets, naisFilesFrom)
	podSpec = envFrom(podSpec, resourceOptions.NativeSecrets, naisEnvFrom)

	return podSpec, err
}

func hostAliases(resourceOptions resource.Options) []corev1.HostAlias {
	var hostAliases []corev1.HostAlias

	for _, hostAlias := range resourceOptions.HostAliases {
		hostAliases = append(hostAliases, corev1.HostAlias{Hostnames: []string{hostAlias.Host}, IP: hostAlias.Address})
	}
	return hostAliases
}

func envFrom(spec *corev1.PodSpec, nativeSecrets bool, naisEnvFrom []nais_io_v1alpha1.EnvFrom) *corev1.PodSpec {
	for _, env := range naisEnvFrom {
		if len(env.ConfigMap) > 0 {
			spec.Containers[0].EnvFrom = append(spec.Containers[0].EnvFrom, fromEnvConfigmap(env.ConfigMap))
		} else if nativeSecrets && len(env.Secret) > 0 {
			spec.Containers[0].EnvFrom = append(spec.Containers[0].EnvFrom, EnvFromSecret(env.Secret))
		}
	}

	return spec
}

func fromEnvConfigmap(name string) corev1.EnvFromSource {
	return corev1.EnvFromSource{
		ConfigMapRef: &corev1.ConfigMapEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: name,
			},
		},
	}
}

func filesFrom(spec *corev1.PodSpec, nativeSecrets bool, naisFilesFrom []nais_io_v1alpha1.FilesFrom) *corev1.PodSpec {
	for _, file := range naisFilesFrom {
		if len(file.ConfigMap) > 0 {
			name := file.ConfigMap
			spec.Volumes = append(spec.Volumes, fromFilesConfigmapVolume(name))
			spec.Containers[0].VolumeMounts = append(spec.Containers[0].VolumeMounts,
				FromFilesVolumeMount(name, file.MountPath, nais_io_v1alpha1.GetDefaultMountPath(name)))
		} else if nativeSecrets && len(file.Secret) > 0 {
			name := file.Secret
			spec.Volumes = append(spec.Volumes, FromFilesSecretVolume(name, name, nil))
			spec.Containers[0].VolumeMounts = append(spec.Containers[0].VolumeMounts,
				FromFilesVolumeMount(name, file.MountPath, nais_io_v1alpha1.DefaultSecretMountPath))
		}
	}

	return spec
}

func fromFilesConfigmapVolume(name string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
			},
		},
	}
}
func podSpecBase(objectMeta metav1.ObjectMeta, options resource.Options, image, preStopHookPath string, appPort int, naisResources nais_io_v1alpha1.ResourceRequirements, livenessProbe, readinessProve, startupProbe *nais_io_v1alpha1.Probe, envVars []nais_io_v1alpha1.EnvVar) *corev1.PodSpec {
	return &corev1.PodSpec{
		Containers:         []corev1.Container{appContainer(objectMeta, options, image, preStopHookPath, appPort, naisResources, livenessProbe, readinessProve, startupProbe, envVars)},
		ServiceAccountName: objectMeta.Name,
		RestartPolicy:      corev1.RestartPolicyAlways,
		DNSPolicy:          corev1.DNSClusterFirst,
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: "gpr-credentials"},
			{Name: "ghcr-credentials"},
		},
	}
}

func appContainer(objectMeta metav1.ObjectMeta, options resource.Options, appImage, preStopHookPath string, appPort int, naisResources nais_io_v1alpha1.ResourceRequirements, livenessProbe, readinessProbe, startupProbe *nais_io_v1alpha1.Probe, naisEnvVars []nais_io_v1alpha1.EnvVar) corev1.Container {
	c := corev1.Container{
		Name:  objectMeta.Name,
		Image: appImage,
		Ports: []corev1.ContainerPort{
			{ContainerPort: int32(appPort), Protocol: corev1.ProtocolTCP, Name: nais_io_v1alpha1.DefaultPortName},
		},
		Resources:       ResourceLimits(naisResources),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Lifecycle:       lifeCycle(preStopHookPath),
		Env:             envVars(objectMeta, options, appImage, naisEnvVars),
	}

	if livenessProbe != nil && len(livenessProbe.Path) > 0 {
		c.LivenessProbe = probe(appPort, *livenessProbe)
	}

	if readinessProbe != nil && len(readinessProbe.Path) > 0 {
		c.ReadinessProbe = probe(appPort, *readinessProbe)
	}

	if startupProbe != nil && len(startupProbe.Path) > 0 {
		c.StartupProbe = probe(appPort, *startupProbe)
	}

	return c
}

func defaultEnvVars(objectMeta metav1.ObjectMeta, options resource.Options, appImage string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: naisAppNameEnv, Value: objectMeta.Name},
		{Name: naisNamespaceEnv, Value: objectMeta.Namespace},
		{Name: naisAppImageEnv, Value: appImage},
		{Name: naisClusterNameEnv, Value: options.ClusterName},
		{Name: naisClientId, Value: AppClientID(objectMeta, options.ClusterName)},
	}
}

// Maps environment variables from ApplicationSpec to the ones we use in CreateSpec
func envVars(objectMeta metav1.ObjectMeta, options resource.Options, appImage string, naisEnvVars []nais_io_v1alpha1.EnvVar) []corev1.EnvVar {
	newEnvVars := defaultEnvVars(objectMeta, options, appImage)

	for _, envVar := range naisEnvVars {
		if envVar.ValueFrom != nil && envVar.ValueFrom.FieldRef.FieldPath != "" {
			newEnvVars = append(newEnvVars, corev1.EnvVar{
				Name: envVar.Name,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{FieldPath: envVar.ValueFrom.FieldRef.FieldPath},
				},
			})
		} else {
			newEnvVars = append(newEnvVars, corev1.EnvVar{Name: envVar.Name, Value: envVar.Value})
		}
	}

	return newEnvVars
}

func ObjectMeta(objectMeta metav1.ObjectMeta, appPort int, prometheusConfig *nais_io_v1alpha1.PrometheusConfig, logFormat, logTransform string) metav1.ObjectMeta {

	port := prometheusConfig.Port
	if len(port) == 0 {
		port = strconv.Itoa(appPort)
	}

	objectMeta.Annotations = map[string]string{}
	if prometheusConfig.Enabled {
		objectMeta.Annotations["prometheus.io/scrape"] = "true"
		objectMeta.Annotations["prometheus.io/port"] = port
		objectMeta.Annotations["prometheus.io/path"] = prometheusConfig.Path
	}

	if len(logFormat) > 0 {
		objectMeta.Annotations["nais.io/logformat"] = logFormat
	}

	if len(logTransform) > 0 {
		objectMeta.Annotations["nais.io/logtransform"] = logTransform
	}

	return objectMeta
}

func lifeCycle(path string) *corev1.Lifecycle {
	if len(path) > 0 {
		return &corev1.Lifecycle{
			PreStop: &corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: strings.TrimLeft(path, "/"),
					Port: intstr.FromString(nais_io_v1alpha1.DefaultPortName),
				},
			},
		}
	}

	return &corev1.Lifecycle{
		PreStop: &corev1.Handler{
			Exec: &corev1.ExecAction{
				Command: []string{"sleep", "5"},
			},
		},
	}
}

func probe(appPort int, probe nais_io_v1alpha1.Probe) *corev1.Probe {
	port := probe.Port
	if port == 0 {
		port = appPort
	}

	k8sprobe := &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: leadingSlash(probe.Path),
				Port: intstr.FromInt(port),
			},
		},
		InitialDelaySeconds: int32(probe.InitialDelay),
		PeriodSeconds:       int32(probe.PeriodSeconds),
		FailureThreshold:    int32(probe.FailureThreshold),
		TimeoutSeconds:      int32(probe.Timeout),
	}

	if probe.Port != 0 {
		k8sprobe.Handler.HTTPGet.Port = intstr.FromInt(probe.Port)
	}

	return k8sprobe
}

func leadingSlash(s string) string {
	if strings.HasPrefix(s, "/") {
		return s
	}
	return "/" + s
}

package pod

import (
	"strconv"
	"strings"

	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/certificateauthority"
	"github.com/nais/naiserator/pkg/resourcecreator/proxyopts"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/securelogs"
	"github.com/spf13/viper"
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

func Spec(app *nais_io_v1alpha1.Application, resourceOptions resource.Options) (*corev1.PodSpec, error) {
	var err error

	podSpec := podSpecBase(app)

	if len(resourceOptions.HostAliases) > 0 {
		podSpec.HostAliases = hostAliases(resourceOptions)
	}

	if !app.Spec.SkipCaBundle {
		podSpec = certificateauthority.CABundle(podSpec)
	}

	podSpec = filesFrom(app, podSpec, resourceOptions.NativeSecrets)

	podSpec = envFrom(app, podSpec, resourceOptions.NativeSecrets)

	if app.Spec.WebProxy && len(resourceOptions.GoogleProjectId) == 0 {
		podSpec, err = proxyopts.ProxyOpts(podSpec)
		if err != nil {
			return nil, err
		}
	}

	if app.Spec.SecureLogs.Enabled {
		podSpec = secureLogs(podSpec)
	}

	return podSpec, err
}

func hostAliases(resourceOptions resource.Options) []corev1.HostAlias {
	var hostAliases []corev1.HostAlias

	for _, hostAlias := range resourceOptions.HostAliases {
		hostAliases = append(hostAliases, corev1.HostAlias{Hostnames: []string{hostAlias.Host}, IP: hostAlias.Address})
	}
	return hostAliases
}

func envFrom(app *nais_io_v1alpha1.Application, spec *corev1.PodSpec, nativeSecrets bool) *corev1.PodSpec {
	for _, env := range app.Spec.EnvFrom {
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

func filesFrom(app *nais_io_v1alpha1.Application, spec *corev1.PodSpec, nativeSecrets bool) *corev1.PodSpec {
	for _, file := range app.Spec.FilesFrom {
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
func podSpecBase(app *nais_io_v1alpha1.Application) *corev1.PodSpec {
	return &corev1.PodSpec{
		Containers:         []corev1.Container{appContainer(app)},
		ServiceAccountName: app.Name,
		RestartPolicy:      corev1.RestartPolicyAlways,
		DNSPolicy:          corev1.DNSClusterFirst,
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: "gpr-credentials"},
			{Name: "ghcr-credentials"},
		},
	}
}

func appContainer(app *nais_io_v1alpha1.Application) corev1.Container {
	c := corev1.Container{
		Name:  app.Name,
		Image: app.Spec.Image,
		Ports: []corev1.ContainerPort{
			{ContainerPort: int32(app.Spec.Port), Protocol: corev1.ProtocolTCP, Name: nais_io_v1alpha1.DefaultPortName},
		},
		Command:         app.Spec.Command,
		Resources:       ResourceLimits(*app.Spec.Resources),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Lifecycle:       lifeCycle(app.Spec.PreStopHookPath),
		Env:             envVars(app),
	}

	if app.Spec.Liveness != nil && len(app.Spec.Liveness.Path) > 0 {
		c.LivenessProbe = probe(app, *app.Spec.Liveness)
	}

	if app.Spec.Readiness != nil && len(app.Spec.Readiness.Path) > 0 {
		c.ReadinessProbe = probe(app, *app.Spec.Readiness)
	}

	if app.Spec.Startup != nil && len(app.Spec.Startup.Path) > 0 {
		c.StartupProbe = probe(app, *app.Spec.Startup)
	}

	return c
}

func secureLogs(podSpec *corev1.PodSpec) *corev1.PodSpec {
	spec := podSpec.DeepCopy()
	spec.Containers = append(spec.Containers, securelogs.FluentdSidecar())
	spec.Containers = append(spec.Containers, securelogs.ConfigmapReloadSidecar())

	spec.Volumes = append(spec.Volumes, securelogs.Volumes()...)

	volumeMount := corev1.VolumeMount{
		Name:      "secure-logs",
		MountPath: "/secure-logs",
	}
	mainContainer := spec.Containers[0].DeepCopy()
	mainContainer.VolumeMounts = append(mainContainer.VolumeMounts, volumeMount)
	spec.Containers[0] = *mainContainer

	return spec
}

func defaultEnvVars(app *nais_io_v1alpha1.Application) []corev1.EnvVar {
	cluster := viper.GetString(config.ClusterName)
	return []corev1.EnvVar{
		{Name: naisAppNameEnv, Value: app.ObjectMeta.Name},
		{Name: naisNamespaceEnv, Value: app.ObjectMeta.Namespace},
		{Name: naisAppImageEnv, Value: app.Spec.Image},
		{Name: naisClusterNameEnv, Value: cluster},
		{Name: naisClientId, Value: app.ClientID(cluster)},
	}
}

// Maps environment variables from ApplicationSpec to the ones we use in Spec
func envVars(app *nais_io_v1alpha1.Application) []corev1.EnvVar {
	newEnvVars := defaultEnvVars(app)

	for _, envVar := range app.Spec.Env {
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

func ObjectMeta(app *nais_io_v1alpha1.Application) metav1.ObjectMeta {
	objectMeta := app.CreateObjectMeta()

	port := app.Spec.Prometheus.Port
	if len(port) == 0 {
		port = strconv.Itoa(app.Spec.Port)
	}

	objectMeta.Annotations = map[string]string{}
	if app.Spec.Prometheus.Enabled {
		objectMeta.Annotations["prometheus.io/scrape"] = "true"
		objectMeta.Annotations["prometheus.io/port"] = port
		objectMeta.Annotations["prometheus.io/path"] = app.Spec.Prometheus.Path
	}

	if len(app.Spec.Logformat) > 0 {
		objectMeta.Annotations["nais.io/logformat"] = app.Spec.Logformat
	}

	if len(app.Spec.Logtransform) > 0 {
		objectMeta.Annotations["nais.io/logtransform"] = app.Spec.Logtransform
	}

	if app.Spec.Elastic != nil {
		objectMeta.Labels["elastic"] = "enabled"
	}

	if app.Spec.Kafka != nil {
		objectMeta.Labels["kafka"] = "enabled"
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

func probe(app *nais_io_v1alpha1.Application, probe nais_io_v1alpha1.Probe) *corev1.Probe {
	port := probe.Port
	if port == 0 {
		port = app.Spec.Port
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

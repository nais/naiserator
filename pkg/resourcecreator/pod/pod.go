package pod

import (
	"strconv"
	"strings"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
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

func reorderContainers(appName string, containers []corev1.Container) []corev1.Container {
	reordered := make([]corev1.Container, len(containers))
	delta := 1
	for i, container := range containers {
		if container.Name == appName {
			reordered[0] = container
			delta = 0
		} else {
			reordered[i+delta] = container
		}
	}
	return reordered
}

func CreateSpec(ast *resource.Ast, resourceOptions resource.Options, appName string, restartPolicy corev1.RestartPolicy) (*corev1.PodSpec, error) {
	var err error

	containers := reorderContainers(appName, ast.Containers)

	podSpec := &corev1.PodSpec{
		InitContainers:     ast.InitContainers,
		Containers:         containers,
		ServiceAccountName: appName,
		RestartPolicy:      restartPolicy,
		DNSPolicy:          corev1.DNSClusterFirst,
		Volumes:            ast.Volumes,
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: "gpr-credentials"},
			{Name: "ghcr-credentials"},
		},
	}

	if len(resourceOptions.HostAliases) > 0 {
		podSpec.HostAliases = hostAliases(resourceOptions)
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

func envFrom(ast *resource.Ast, nativeSecrets bool, naisEnvFrom []nais_io_v1.EnvFrom) {
	for _, env := range naisEnvFrom {
		if len(env.ConfigMap) > 0 {
			ast.EnvFrom = append(ast.EnvFrom, fromEnvConfigmap(env.ConfigMap))
		} else if nativeSecrets && len(env.Secret) > 0 {
			ast.EnvFrom = append(ast.EnvFrom, EnvFromSecret(env.Secret))
		}
	}
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

func filesFrom(ast *resource.Ast, nativeSecrets bool, naisFilesFrom []nais_io_v1.FilesFrom) {
	for _, file := range naisFilesFrom {
		if len(file.ConfigMap) > 0 {
			name := file.ConfigMap
			ast.Volumes = append(ast.Volumes, fromFilesConfigmapVolume(name))
			ast.VolumeMounts = append(ast.VolumeMounts,
				FromFilesVolumeMount(name, file.MountPath, nais_io_v1alpha1.GetDefaultMountPath(name)))
		} else if nativeSecrets && len(file.Secret) > 0 {
			name := file.Secret
			ast.Volumes = append(ast.Volumes, FromFilesSecretVolume(name, name, nil))
			ast.VolumeMounts = append(ast.VolumeMounts, FromFilesVolumeMount(name, file.MountPath, nais_io_v1alpha1.DefaultSecretMountPath))
		}
	}
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

func CreateAppContainer(app *nais_io_v1alpha1.Application, ast *resource.Ast, options resource.Options) {
	ast.Env = append(ast.Env, app.Spec.Env.ToKubernetes()...)
	ast.Env = append(ast.Env, defaultEnvVars(app, options.ClusterName, app.Spec.Image)...)
	filesFrom(ast, options.NativeSecrets, app.Spec.FilesFrom)
	envFrom(ast, options.NativeSecrets, app.Spec.EnvFrom)

	container := corev1.Container{
		Name:  app.Name,
		Image: app.Spec.Image,
		Ports: []corev1.ContainerPort{
			{ContainerPort: int32(app.Spec.Port), Protocol: corev1.ProtocolTCP, Name: nais_io_v1alpha1.DefaultPortName},
		},
		Command:         app.Spec.Command,
		Resources:       ResourceLimits(*app.Spec.Resources),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Lifecycle:       lifeCycle(app.Spec.PreStopHookPath),
		Env:             ast.Env,
		EnvFrom:         ast.EnvFrom,
		VolumeMounts:    ast.VolumeMounts,
	}

	if app.Spec.Liveness != nil && len(app.Spec.Liveness.Path) > 0 {
		container.LivenessProbe = probe(app.Spec.Port, *app.Spec.Liveness)
	}

	if app.Spec.Readiness != nil && len(app.Spec.Readiness.Path) > 0 {
		container.ReadinessProbe = probe(app.Spec.Port, *app.Spec.Readiness)
	}

	if app.Spec.Startup != nil && len(app.Spec.Startup.Path) > 0 {
		container.StartupProbe = probe(app.Spec.Port, *app.Spec.Startup)
	}

	ast.Containers = append(ast.Containers, container)
}

func CreateNaisjobContainer(naisjob *nais_io_v1.Naisjob, ast *resource.Ast, options resource.Options) {
	ast.Env = append(ast.Env, naisjob.Spec.Env.ToKubernetes()...)
	ast.Env = append(ast.Env, defaultEnvVars(naisjob, options.ClusterName, naisjob.Spec.Image)...)
	filesFrom(ast, options.NativeSecrets, naisjob.Spec.FilesFrom)
	envFrom(ast, options.NativeSecrets, naisjob.Spec.EnvFrom)

	container := corev1.Container{
		Name:            naisjob.Name,
		Image:           naisjob.Spec.Image,
		Command:         naisjob.Spec.Command,
		Resources:       ResourceLimits(*naisjob.Spec.Resources),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Lifecycle:       lifeCycle(naisjob.Spec.PreStopHookPath),
		Env:             ast.Env,
		EnvFrom:         ast.EnvFrom,
		VolumeMounts:    ast.VolumeMounts,
	}

	ast.Containers = append(ast.Containers, container)
}

func defaultEnvVars(source resource.Source, clusterName, appImage string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: naisAppNameEnv, Value: source.GetName()},
		{Name: naisNamespaceEnv, Value: source.GetNamespace()},
		{Name: naisAppImageEnv, Value: appImage},
		{Name: naisClusterNameEnv, Value: clusterName},
		{Name: naisClientId, Value: AppClientID(source, clusterName)},
	}
}

func mapMerge(dst, src map[string]string) {
	for k, v := range src {
		dst[k] = v
	}
}

func CreateAppObjectMeta(app *nais_io_v1alpha1.Application, ast *resource.Ast) metav1.ObjectMeta {
	objectMeta := resource.CreateObjectMeta(app)
	objectMeta.Annotations = ast.Annotations
	mapMerge(objectMeta.Labels, ast.Labels)

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

	return objectMeta
}

func CreateNaisjobObjectMeta(naisjob *nais_io_v1.Naisjob, ast *resource.Ast) metav1.ObjectMeta {
	objectMeta := resource.CreateObjectMeta(naisjob)
	objectMeta.Annotations = ast.Annotations
	mapMerge(objectMeta.Labels, ast.Labels)

	objectMeta.Annotations = map[string]string{}

	if len(naisjob.Spec.Logformat) > 0 {
		objectMeta.Annotations["nais.io/logformat"] = naisjob.Spec.Logformat
	}

	if len(naisjob.Spec.Logtransform) > 0 {
		objectMeta.Annotations["nais.io/logtransform"] = naisjob.Spec.Logtransform
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

func probe(appPort int, probe nais_io_v1.Probe) *corev1.Probe {
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

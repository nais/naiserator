package pod

import (
	"fmt"
	"maps"
	"regexp"
	"strconv"
	"strings"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

const (
	naisAppImageEnv    = "NAIS_APP_IMAGE"
	naisAppNameEnv     = "NAIS_APP_NAME"
	naisClientId       = "NAIS_CLIENT_ID"
	naisClusterNameEnv = "NAIS_CLUSTER_NAME"
	naisNamespaceEnv   = "NAIS_NAMESPACE"
	naisPodNameEnv     = "NAIS_POD_NAME"
	defaultPort        = "8080"
)

type EnvSource interface {
	resource.Source
	GetEnv() nais_io_v1.EnvVars
	GetEffectiveImage() string
}

type Source interface {
	resource.Source
	GetCommand() []string
	GetEnv() nais_io_v1.EnvVars
	GetEnvFrom() []nais_io_v1.EnvFrom
	GetFilesFrom() []nais_io_v1.FilesFrom
	GetEffectiveImage() string
	GetLiveness() *nais_io_v1.Probe
	GetLogformat() string
	GetLogtransform() string
	GetPort() int
	GetPreStopHook() *nais_io_v1.PreStopHook
	GetPreStopHookPath() string
	GetPrometheus() *nais_io_v1.PrometheusConfig
	GetReadiness() *nais_io_v1.Probe
	GetResources() *nais_io_v1.ResourceRequirements
	GetStartup() *nais_io_v1.Probe
}

type Config interface {
	GetClusterName() string
	GetGoogleProjectID() string
	GetGoogleTeamProjectID() string
	GetHostAliases() []config.HostAlias
	GetImagePullSecrets() []string
	IsGCPEnabled() bool
	IsPrometheusOperatorEnabled() bool
	IsGARTolerationEnabled() bool
}

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

func CreateSpec(ast *resource.Ast, cfg Config, appName string, annotations map[string]string, restartPolicy corev1.RestartPolicy, terminationGracePeriodSeconds *int64) (*corev1.PodSpec, error) {
	if len(ast.Containers) == 0 {
		return &corev1.PodSpec{}, nil
	}

	containers := reorderContainers(appName, ast.Containers)

	// Pod security context will by default make the filesystem read-only. Mount an emptyDir on /tmp
	// to allow temporary files to be created.
	volumes := append(ast.Volumes, corev1.Volume{
		Name: "writable-tmp",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	containers[0].VolumeMounts = append(containers[0].VolumeMounts, corev1.VolumeMount{
		Name:      "writable-tmp",
		MountPath: "/tmp",
	})

	tolerations := SetupTolerations(cfg, containers[0].Image)
	affinity := ConfigureAffinity(appName, tolerations)

	podSpec := &corev1.PodSpec{
		InitContainers:                ast.InitContainers,
		Containers:                    containers,
		ServiceAccountName:            appName,
		RestartPolicy:                 restartPolicy,
		DNSPolicy:                     corev1.DNSClusterFirst,
		Volumes:                       volumes,
		ImagePullSecrets:              imagePullSecrets(cfg),
		TerminationGracePeriodSeconds: terminationGracePeriodSeconds,
		Affinity:                      affinity,
		Tolerations:                   tolerations,
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup:             ptr.To[int64](1069),
			FSGroupChangePolicy: ptr.To(corev1.FSGroupChangeOnRootMismatch),
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		},
	}

	podSpec.Containers[0].SecurityContext = configureSecurityContext(annotations)

	if len(cfg.GetHostAliases()) > 0 {
		podSpec.HostAliases = hostAliases(cfg)
	}

	return podSpec, nil
}

func DefaultContainerSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: ptr.To(false),
		Privileged:               ptr.To(false),
		ReadOnlyRootFilesystem:   ptr.To(true),
		RunAsNonRoot:             ptr.To(true),
		RunAsGroup:               ptr.To(int64(1069)),
		RunAsUser:                ptr.To(int64(1069)),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}

func configureSecurityContext(annotations map[string]string) *corev1.SecurityContext {
	ctx := DefaultContainerSecurityContext()
	ctx.RunAsUser = ptr.To(runAsUser(annotations))
	ctx.RunAsGroup = ptr.To(runAsGroup(annotations))
	ctx.ReadOnlyRootFilesystem = ptr.To(readOnlyFileSystem(annotations))
	return ctx
}

func runAsUser(annotations map[string]string) int64 {
	val, found := annotations["nais.io/run-as-user"]
	if !found {
		return 1069
	}

	uid, err := strconv.Atoi(val)
	if err != nil {
		log.Warnf("Converting string to int: %v", err)
		return 1069
	}

	return int64(uid)
}

func runAsGroup(annotations map[string]string) int64 {
	val, found := annotations["nais.io/run-as-group"]
	if !found {
		return runAsUser(annotations)
	}

	uid, err := strconv.Atoi(val)
	if err != nil {
		log.Warnf("Converting string to int: %v", err)
		return runAsUser(annotations)
	}

	return int64(uid)
}

func hostAliases(cfg Config) []corev1.HostAlias {
	var hostAliases []corev1.HostAlias

	for _, hostAlias := range cfg.GetHostAliases() {
		hostAliases = append(hostAliases, corev1.HostAlias{Hostnames: []string{hostAlias.Host}, IP: hostAlias.Address})
	}
	return hostAliases
}

func envFrom(ast *resource.Ast, naisEnvFrom []nais_io_v1.EnvFrom) {
	for _, env := range naisEnvFrom {
		if len(env.ConfigMap) > 0 {
			ast.EnvFrom = append(ast.EnvFrom, fromEnvConfigmap(env.ConfigMap))
		} else if len(env.Secret) > 0 {
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

// A lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-',
// and must start and end with an alphanumeric character
var regNameNormalizer = regexp.MustCompile("[^a-z0-9-]+")

func generateNameFromMountPath(mountPath string) string {
	s := strings.ToLower(mountPath)
	s = regNameNormalizer.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func filesFrom(ast *resource.Ast, naisFilesFrom []nais_io_v1.FilesFrom) {
	for _, file := range naisFilesFrom {
		if len(file.ConfigMap) > 0 {
			name := file.ConfigMap
			ast.Volumes = append(ast.Volumes, fromFilesConfigmapVolume(name))
			ast.VolumeMounts = append(ast.VolumeMounts,
				FromFilesVolumeMount(name, file.MountPath, nais_io_v1alpha1.GetDefaultMountPath(name), true))
		} else if len(file.Secret) > 0 {
			name := file.Secret
			ast.Volumes = append(ast.Volumes, FromFilesSecretVolume(name, name, nil))
			ast.VolumeMounts = append(ast.VolumeMounts, FromFilesVolumeMount(name, file.MountPath, nais_io_v1alpha1.DefaultSecretMountPath, true))
		} else if len(file.PersistentVolumeClaim) > 0 {
			name := file.PersistentVolumeClaim
			ast.Volumes = append(ast.Volumes, FromFilesPVCVolume(name, name))
			ast.VolumeMounts = append(ast.VolumeMounts, FromFilesVolumeMount(name, file.MountPath, nais_io_v1alpha1.GetDefaultPVCMountPath(name), false))
		} else if file.EmptyDir != nil && len(file.MountPath) > 0 {
			name := generateNameFromMountPath(file.MountPath)
			ast.Volumes = append(ast.Volumes, FilesFromEmptyDir(name, file.EmptyDir.Medium))
			ast.VolumeMounts = append(ast.VolumeMounts, FromFilesVolumeMount(name, file.MountPath, file.MountPath, false))
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

func imagePullSecrets(cfg Config) []corev1.LocalObjectReference {
	var secrets []corev1.LocalObjectReference
	for _, secret := range cfg.GetImagePullSecrets() {
		secrets = append(secrets, corev1.LocalObjectReference{Name: secret})
	}
	return secrets
}

func CreateContainerEnvVars(app EnvSource, ast *resource.Ast, cfg Config) {
	ast.Env = append(ast.Env, defaultEnvVars(app, cfg, app.GetEffectiveImage())...)
	ast.Env = append(ast.Env, app.GetEnv().ToKubernetes()...)
}

func CreateAppContainer(app Source, ast *resource.Ast, cfg Config) error {
	filesFrom(ast, app.GetFilesFrom())
	envFrom(ast, app.GetEnvFrom())
	lifecycle, err := lifecycle(app.GetPreStopHookPath(), app.GetPreStopHook())
	if err != nil {
		return err
	}

	containerPorts := []corev1.ContainerPort{
		{ContainerPort: int32(app.GetPort()), Protocol: corev1.ProtocolTCP, Name: nais_io_v1alpha1.DefaultPortName},
	}

	if cfg.IsPrometheusOperatorEnabled() {
		if app.GetPrometheus().Port != "" {
			promPort, err := strconv.ParseInt(app.GetPrometheus().Port, 10, 32)
			if err != nil {
				return fmt.Errorf("invalid port provided, unable to convert to int32")
			}

			if promPort != 0 && int(promPort) != app.GetPort() {
				containerPorts = append(containerPorts, corev1.ContainerPort{
					ContainerPort: int32(promPort),
					Protocol:      corev1.ProtocolTCP,
					Name:          "metrics",
				})
			}
		}
	}

	container := corev1.Container{
		Name:            app.GetName(),
		Image:           app.GetEffectiveImage(),
		Ports:           containerPorts,
		Command:         app.GetCommand(),
		Resources:       ResourceLimits(*app.GetResources()),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Lifecycle:       lifecycle,
		Env:             ast.Env,
		EnvFrom:         ast.EnvFrom,
		VolumeMounts:    ast.VolumeMounts,
	}

	if app.GetLiveness() != nil && len(app.GetLiveness().Path) > 0 {
		container.LivenessProbe = probe(app.GetPort(), *app.GetLiveness())
	}

	if app.GetReadiness() != nil && len(app.GetReadiness().Path) > 0 {
		container.ReadinessProbe = probe(app.GetPort(), *app.GetReadiness())
	}

	if app.GetStartup() != nil && len(app.GetStartup().Path) > 0 {
		container.StartupProbe = probe(app.GetPort(), *app.GetStartup())
	}

	ast.Containers = append(ast.Containers, container)

	return nil
}

func CreateNaisjobContainer(naisjob *nais_io_v1.Naisjob, ast *resource.Ast, cfg Config) error {
	filesFrom(ast, naisjob.Spec.FilesFrom)
	envFrom(ast, naisjob.Spec.EnvFrom)
	lifecycle, err := lifecycle("", naisjob.Spec.PreStopHook)
	if err != nil {
		return err
	}

	container := corev1.Container{
		Name:            naisjob.Name,
		Image:           naisjob.GetEffectiveImage(),
		Command:         naisjob.Spec.Command,
		Resources:       ResourceLimits(*naisjob.Spec.Resources),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Lifecycle:       lifecycle,
		Env:             ast.Env,
		EnvFrom:         ast.EnvFrom,
		VolumeMounts:    ast.VolumeMounts,
	}

	ast.Containers = append(ast.Containers, container)

	return err
}

func defaultEnvVars(source resource.Source, cfg Config, appImage string) []corev1.EnvVar {
	port := strconv.Itoa(source.GetPort())
	if source.GetPort() == 0 {
		port = defaultPort
	}

	podName := &corev1.EnvVarSource{
		FieldRef: &corev1.ObjectFieldSelector{
			FieldPath: "metadata.name",
		},
	}

	envs := []corev1.EnvVar{
		{Name: naisAppNameEnv, Value: source.GetName()},
		{Name: naisNamespaceEnv, Value: source.GetNamespace()},
		{Name: naisAppImageEnv, Value: appImage},
		{Name: naisClusterNameEnv, Value: cfg.GetClusterName()},
		{Name: naisClientId, Value: AppClientID(source, cfg.GetClusterName())},
		{Name: naisPodNameEnv, ValueFrom: podName},
		{Name: "LOG4J_FORMAT_MSG_NO_LOOKUPS", Value: "true"},
		{Name: "PORT", Value: port},
		{Name: "BIND_ADDRESS", Value: "0.0.0.0:" + port},
	}

	if googleTeamProjectID := cfg.GetGoogleTeamProjectID(); len(googleTeamProjectID) > 0 {
		envs = append(envs,
			corev1.EnvVar{Name: "GOOGLE_CLOUD_PROJECT", Value: googleTeamProjectID},
			corev1.EnvVar{Name: "GCP_TEAM_PROJECT_ID", Value: googleTeamProjectID}, // Legacy variable
		)
	}

	return envs
}

func CreateAppObjectMeta(app Source, ast *resource.Ast, cfg Config) metav1.ObjectMeta {
	objectMeta := resource.CreateObjectMeta(app)
	objectMeta.Annotations = ast.Annotations
	maps.Copy(objectMeta.Labels, ast.Labels)

	port := app.GetPrometheus().Port
	if len(port) == 0 {
		port = strconv.Itoa(app.GetPort())
	}

	objectMeta.Annotations["kubectl.kubernetes.io/default-container"] = app.GetName()

	if cfg.IsGCPEnabled() {
		objectMeta.Annotations["cluster-autoscaler.kubernetes.io/safe-to-evict"] = "true"
	}

	if app.GetPrometheus().Enabled {
		objectMeta.Annotations["prometheus.io/scrape"] = "true"
		objectMeta.Annotations["prometheus.io/port"] = port
		objectMeta.Annotations["prometheus.io/path"] = app.GetPrometheus().Path
	}

	if len(app.GetLogformat()) > 0 {
		objectMeta.Annotations["nais.io/logformat"] = app.GetLogformat()
	}

	if len(app.GetLogtransform()) > 0 {
		objectMeta.Annotations["nais.io/logtransform"] = app.GetLogtransform()
	}

	return objectMeta
}

func CreateNaisjobObjectMeta(naisjob *nais_io_v1.Naisjob, ast *resource.Ast, cfg Config) metav1.ObjectMeta {
	objectMeta := resource.CreateObjectMeta(naisjob)
	objectMeta.Annotations = ast.Annotations
	maps.Copy(objectMeta.Labels, ast.Labels)

	objectMeta.Annotations["kubectl.kubernetes.io/default-container"] = naisjob.GetName()

	// enables HAHAHA
	objectMeta.Labels["nais.io/naisjob"] = "true"

	if len(naisjob.Spec.Logformat) > 0 {
		objectMeta.Annotations["nais.io/logformat"] = naisjob.Spec.Logformat
	}

	if len(naisjob.Spec.Logtransform) > 0 {
		objectMeta.Annotations["nais.io/logtransform"] = naisjob.Spec.Logtransform
	}

	return objectMeta
}

// The default PreStopHook will wait for five seconds before killing pods.
// Earlier, this function was solved by "sleep 5" inside the container.
func defaultPreStopHook() *corev1.Lifecycle {
	return &corev1.Lifecycle{
		PreStop: &corev1.LifecycleHandler{
			Sleep: &corev1.SleepAction{
				Seconds: 5,
			},
		},
	}
}

// lifecycle creates lifecycle definitions, right now adding only PreStop handlers.
//
// preStopHookPath is the old, deprecated way of adding preStopHook definitions.
// This function handles both of them.
func lifecycle(preStopHookPath string, preStopHook *nais_io_v1.PreStopHook) (*corev1.Lifecycle, error) {
	if len(preStopHookPath) > 0 && preStopHook != nil {
		return nil, fmt.Errorf("can only use one of spec.preStopHookPath or spec.preStopHook")
	}

	// Legacy behavior.
	if len(preStopHookPath) > 0 {
		return &corev1.Lifecycle{
			PreStop: &corev1.LifecycleHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: strings.TrimLeft(preStopHookPath, "/"),
					Port: intstr.FromString(nais_io_v1alpha1.DefaultPortName),
				},
			},
		}, nil
	}

	if preStopHook == nil {
		return defaultPreStopHook(), nil
	}

	if preStopHook.Exec != nil && preStopHook.Http != nil {
		return nil, fmt.Errorf("can only use one type of preStopHook, either exec or http")
	}

	if preStopHook.Exec != nil {
		return &corev1.Lifecycle{
			PreStop: &corev1.LifecycleHandler{
				Exec: &corev1.ExecAction{
					Command: preStopHook.Exec.Command,
				},
			},
		}, nil
	}

	var port intstr.IntOrString
	if preStopHook.Http.Port == nil {
		port = intstr.FromString(nais_io_v1alpha1.DefaultPortName)
	} else {
		port = intstr.FromInt(*preStopHook.Http.Port)
	}

	return &corev1.Lifecycle{
		PreStop: &corev1.LifecycleHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: preStopHook.Http.Path,
				Port: port,
			},
		},
	}, nil
}

func probe(appPort int, probe nais_io_v1.Probe) *corev1.Probe {
	port := probe.Port
	if port == 0 {
		port = appPort
	}

	k8sprobe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
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
		k8sprobe.HTTPGet.Port = intstr.FromInt(probe.Port)
	}

	return k8sprobe
}

func leadingSlash(s string) string {
	if strings.HasPrefix(s, "/") {
		return s
	}
	return "/" + s
}

func readOnlyFileSystem(annotations map[string]string) bool {
	val, found := annotations["nais.io/read-only-file-system"]
	if !found {
		return true
	}

	return strings.ToLower(val) != "false"
}

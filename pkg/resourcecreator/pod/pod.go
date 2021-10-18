package pod

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"k8s.io/utils/pointer"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

const (
	naisAppNameEnv     = "NAIS_APP_NAME"
	naisNamespaceEnv   = "NAIS_NAMESPACE"
	naisAppImageEnv    = "NAIS_APP_IMAGE"
	naisClusterNameEnv = "NAIS_CLUSTER_NAME"
	naisClientId       = "NAIS_CLIENT_ID"
)

type Source interface {
	resource.Source
	GetCommand() []string
	GetEnv() nais_io_v1.EnvVars
	GetEnvFrom() []nais_io_v1.EnvFrom
	GetFilesFrom() []nais_io_v1.FilesFrom
	GetImage() string
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

func CreateSpec(ast *resource.Ast, resourceOptions resource.Options, appName string, annotations map[string]string, restartPolicy corev1.RestartPolicy) (*corev1.PodSpec, error) {
	var err error

	containers := reorderContainers(appName, ast.Containers)

	// Pod security context will by default make the filesystem read-only. Mount an emptyDir on /tmp
	// to allow temporary files to be created.
	volumes := append(ast.Volumes, corev1.Volume{
		Name: "writable-tmp",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	if len(containers) > 0 {
		containers[0].VolumeMounts = append(containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      "writable-tmp",
			MountPath: "/tmp",
		})
	}

	podSpec := &corev1.PodSpec{
		InitContainers:     ast.InitContainers,
		Containers:         containers,
		ServiceAccountName: appName,
		RestartPolicy:      restartPolicy,
		DNSPolicy:          corev1.DNSClusterFirst,
		Volumes:            volumes,
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: "gh-docker-credentials"},
		},
	}

	if resourceOptions.SecurePodSecurityContext && !exploitable(annotations) { // TODO(jhrv): remove SecurePodSecurityContext option all together when this is rolled out in all clusters
		podSpec.Containers[0].SecurityContext = &corev1.SecurityContext{
			RunAsUser:                pointer.Int64Ptr(runAsUser(annotations)),
			RunAsGroup:               pointer.Int64Ptr(runAsGroup(annotations)),
			RunAsNonRoot:             pointer.BoolPtr(true),
			Privileged:               pointer.BoolPtr(false),
			AllowPrivilegeEscalation: pointer.BoolPtr(false),
			ReadOnlyRootFilesystem:   pointer.BoolPtr(readOnlyFileSystem(annotations)),
		}

		capabilities := &corev1.Capabilities{
			Drop: []corev1.Capability{"all"},
		}

		additionalCapabilities := sanitizeCapabilities(annotations, resourceOptions.AllowedKernelCapabilities)
		if additionalCapabilities != nil && len(additionalCapabilities) > 0 {
			capabilities.Add = additionalCapabilities
		}

		podSpec.Containers[0].SecurityContext.Capabilities = capabilities
	}

	if len(resourceOptions.HostAliases) > 0 {
		podSpec.HostAliases = hostAliases(resourceOptions)
	}

	return podSpec, err
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
				FromFilesVolumeMount(name, file.MountPath, nais_io_v1alpha1.GetDefaultMountPath(name), true))
		} else if nativeSecrets && len(file.Secret) > 0 {
			name := file.Secret
			ast.Volumes = append(ast.Volumes, FromFilesSecretVolume(name, name, nil))
			ast.VolumeMounts = append(ast.VolumeMounts, FromFilesVolumeMount(name, file.MountPath, nais_io_v1alpha1.DefaultSecretMountPath, true))
		} else if len(file.PersistentVolumeClaim) > 0 {
			name := file.PersistentVolumeClaim
			ast.Volumes = append(ast.Volumes, FromFilesPVCVolume(name, name))
			ast.VolumeMounts = append(ast.VolumeMounts, FromFilesVolumeMount(name, file.MountPath, nais_io_v1alpha1.GetDefaultPVCMountPath(name), false))
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

func CreateAppContainer(app Source, ast *resource.Ast, options resource.Options) error {
	ast.Env = append(ast.Env, app.GetEnv().ToKubernetes()...)
	ast.Env = append(ast.Env, defaultEnvVars(app, options.ClusterName, app.GetImage())...)
	filesFrom(ast, options.NativeSecrets, app.GetFilesFrom())
	envFrom(ast, options.NativeSecrets, app.GetEnvFrom())
	lifecycle, err := lifecycle(app.GetPreStopHookPath(), app.GetPreStopHook())
	if err != nil {
		return err
	}

	container := corev1.Container{
		Name:  app.GetName(),
		Image: app.GetImage(),
		Ports: []corev1.ContainerPort{
			{ContainerPort: int32(app.GetPort()), Protocol: corev1.ProtocolTCP, Name: nais_io_v1alpha1.DefaultPortName},
		},
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

func CreateNaisjobContainer(naisjob *nais_io_v1.Naisjob, ast *resource.Ast, options resource.Options) error {
	ast.Env = append(ast.Env, naisjob.Spec.Env.ToKubernetes()...)
	ast.Env = append(ast.Env, defaultEnvVars(naisjob, options.ClusterName, naisjob.Spec.Image)...)
	filesFrom(ast, options.NativeSecrets, naisjob.Spec.FilesFrom)
	envFrom(ast, options.NativeSecrets, naisjob.Spec.EnvFrom)
	lifecycle, err := lifecycle("", naisjob.Spec.PreStopHook)
	if err != nil {
		return err
	}

	container := corev1.Container{
		Name:            naisjob.Name,
		Image:           naisjob.Spec.Image,
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

func CreateAppObjectMeta(app Source, ast *resource.Ast, opt *resource.Options) metav1.ObjectMeta {
	objectMeta := resource.CreateObjectMeta(app)
	objectMeta.Annotations = ast.Annotations
	mapMerge(objectMeta.Labels, ast.Labels)

	port := app.GetPrometheus().Port
	if len(port) == 0 {
		port = strconv.Itoa(app.GetPort())
	}

	objectMeta.Annotations = map[string]string{}
	if len(opt.GoogleProjectId) > 0 {
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

	if opt.Linkerd {
		copyLinkerdAnnotations(app.GetAnnotations(), objectMeta.Annotations)
	}

	return objectMeta
}

func CreateNaisjobObjectMeta(naisjob *nais_io_v1.Naisjob, ast *resource.Ast, opt *resource.Options) metav1.ObjectMeta {
	objectMeta := resource.CreateObjectMeta(naisjob)
	objectMeta.Annotations = ast.Annotations
	mapMerge(objectMeta.Labels, ast.Labels)

	objectMeta.Annotations = map[string]string{}

	objectMeta.Annotations["ginuudan.nais.io/dwindle"] = "true"

	if len(naisjob.Spec.Logformat) > 0 {
		objectMeta.Annotations["nais.io/logformat"] = naisjob.Spec.Logformat
	}

	if len(naisjob.Spec.Logtransform) > 0 {
		objectMeta.Annotations["nais.io/logtransform"] = naisjob.Spec.Logtransform
	}

	if opt.Linkerd {
		copyLinkerdAnnotations(naisjob.Annotations, objectMeta.Annotations)
	}

	return objectMeta
}

func copyLinkerdAnnotations(src, dst map[string]string) {
	for k, v := range src {
		if strings.HasPrefix(k, "config.linkerd.io/") {
			dst[k] = v
		}
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
			PreStop: &corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: strings.TrimLeft(preStopHookPath, "/"),
					Port: intstr.FromString(nais_io_v1alpha1.DefaultPortName),
				},
			},
		}, nil
	}

	if preStopHook == nil {
		return &corev1.Lifecycle{
			PreStop: &corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{"sleep", "5"},
				},
			},
		}, nil
	}

	if preStopHook.Exec != nil && preStopHook.Http != nil {
		return nil, fmt.Errorf("can only use one type of preStopHook, either exec or http")
	}

	if preStopHook.Exec != nil {
		return &corev1.Lifecycle{
			PreStop: &corev1.Handler{
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
		PreStop: &corev1.Handler{
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

func exploitable(annotations map[string]string) bool {
	val, found := annotations["nais.io/security-does-not-matter"]
	if !found {
		return false
	}

	return strings.ToLower(val) == "true"
}

func readOnlyFileSystem(annotations map[string]string) bool {
	val, found := annotations["nais.io/read-only-file-system"]
	if !found {
		return true
	}

	return strings.ToLower(val) != "false"
}

func sanitizeCapabilities(annotations map[string]string, allowedCapabilites []string) []corev1.Capability {
	val, found := annotations["nais.io/add-kernel-capability"]
	if !found {
		return nil
	}

	capabilities := make([]corev1.Capability, 0)
	desiredCapabilites := strings.Split(val, ",")
	for _, desiredCapability := range desiredCapabilites {
		if allowed(desiredCapability, allowedCapabilites) {
			capabilities = append(capabilities, corev1.Capability(strings.ToUpper(desiredCapability)))
		}
	}

	return capabilities
}

func allowed(capability string, allowedCapabilites []string) bool {
	for _, allowedCapability := range allowedCapabilites {
		if strings.ToLower(capability) == strings.ToLower(allowedCapability) {
			return true
		}
	}
	return false
}

package resourcecreator

import (
	"fmt"
	"strconv"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/securelogs"
	"github.com/nais/naiserator/pkg/vault"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Deployment(app *nais.Application, resourceOptions ResourceOptions) (*appsv1.Deployment, error) {
	spec, err := deploymentSpec(app, resourceOptions)
	if err != nil {
		return nil, err
	}

	objectMeta := app.CreateObjectMeta()
	if val, ok := app.Annotations["kubernetes.io/change-cause"]; ok {
		if objectMeta.Annotations == nil {
			objectMeta.Annotations = make(map[string]string)
		}

		objectMeta.Annotations["kubernetes.io/change-cause"] = val
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: objectMeta,
		Spec:       *spec,
	}, nil
}

func deploymentSpec(app *nais.Application, resourceOptions ResourceOptions) (*appsv1.DeploymentSpec, error) {
	podSpec, err := podSpec(resourceOptions, app)
	if err != nil {
		return nil, err
	}

	var strategy appsv1.DeploymentStrategy
	if app.Spec.Strategy.Type == nais.DeploymentStrategyRecreate {
		strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		}
	} else if app.Spec.Strategy.Type == nais.DeploymentStrategyRollingUpdate {
		strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDeployment{
				MaxUnavailable: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: int32(0),
				},
				MaxSurge: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: int32(1),
				},
			},
		}
	}

	return &appsv1.DeploymentSpec{
		Replicas: int32p(resourceOptions.NumReplicas),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": app.Name},
		},
		Strategy:                strategy,
		ProgressDeadlineSeconds: int32p(300),
		RevisionHistoryLimit:    int32p(10),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: podObjectMeta(app),
			Spec:       *podSpec,
		},
	}, nil
}

func podSpec(resourceOptions ResourceOptions, app *nais.Application) (*corev1.PodSpec, error) {
	var err error

	ApplySecretDefaults(&app.Spec.Secrets)

	podSpec := podSpecBase(app)

	if resourceOptions.NativeSecrets && len(app.Spec.Secrets) > 0 {
		podSpec = secrets(app, podSpec)
	}

	if app.Spec.LeaderElection {
		podSpec = leaderElection(app, podSpec)
	}

	if !app.Spec.SkipCaBundle {
		podSpec = caBundle(podSpec)
	}

	podSpec = configMapFiles(app, podSpec)

	if vault.Enabled() && app.Spec.Vault.Enabled {
		podSpec, err = vaultContainers(app, podSpec)
		if err != nil {
			return nil, err
		}
	}

	if app.Spec.WebProxy {
		podSpec, err = proxyOpts(podSpec)
		if err != nil {
			return nil, err
		}
	}

	if app.Spec.SecureLogs.Enabled {
		podSpec = secureLogs(podSpec)
	}

	return podSpec, err
}

func secrets(application *nais.Application, podSpecRef *corev1.PodSpec) *corev1.PodSpec {
	spec := podSpecRef.DeepCopy()
	appContainer := GetContainerByName(spec.Containers, application.Name)

	for _, s := range application.Spec.Secrets {
		if s.Type == nais.SecretTypeEnv {
			appContainer.EnvFrom = append(appContainer.EnvFrom, envFromSecret(s))
			continue
		}

		if s.Type == nais.SecretTypeFiles {
			spec.Volumes = append(spec.Volumes, secretVolume(s))
			appContainer.VolumeMounts = append(appContainer.VolumeMounts, secretVolumeMount(s))
			continue
		}
	}

	return spec
}

func secretVolumeMount(secret nais.Secret) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      secret.Name,
		ReadOnly:  true,
		MountPath: secret.MountPath}
}

func secretVolume(secret nais.Secret) corev1.Volume {
	return corev1.Volume{
		Name: secret.Name,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secret.Name}}}
}

func envFromSecret(secret nais.Secret) corev1.EnvFromSource {
	return corev1.EnvFromSource{
		SecretRef: &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: secret.Name,
			}}}
}

func ApplySecretDefaults(secretsRef *[]nais.Secret) {
	secrets := *secretsRef
	for i, secret := range secrets {
		if len(secret.Type) == 0 {
			secrets[i].Type = nais.DefaultSecretType
		}

		if secret.Type == nais.SecretTypeFiles && len(secret.MountPath) == 0 {
			secrets[i].MountPath = nais.DefaultSecretMountPath
		}
	}
}

func configMapFiles(app *nais.Application, spec *corev1.PodSpec) *corev1.PodSpec {
	for _, cm := range app.Spec.ConfigMaps.Files {
		volumeName := fmt.Sprintf("nais-cm-%s", cm)

		volume := corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cm,
					},
				},
			},
		}

		volumeMount := corev1.VolumeMount{
			Name:      volumeName,
			MountPath: fmt.Sprintf("/var/run/configmaps/%s", cm),
		}

		spec.Volumes = append(spec.Volumes, volume)
		spec.Containers[0].VolumeMounts = append(spec.Containers[0].VolumeMounts, volumeMount)
	}

	return spec
}

func podSpecBase(app *nais.Application) *corev1.PodSpec {
	return &corev1.PodSpec{
		Containers:         []corev1.Container{appContainer(app)},
		ServiceAccountName: app.Name,
		RestartPolicy:      corev1.RestartPolicyAlways,
		DNSPolicy:          corev1.DNSClusterFirst,
		ImagePullSecrets:   []corev1.LocalObjectReference{{Name: "gpr-credentials"}},
	}
}

func appContainer(app *nais.Application) corev1.Container {
	c := corev1.Container{
		Name:  app.Name,
		Image: app.Spec.Image,
		Ports: []corev1.ContainerPort{
			{ContainerPort: int32(app.Spec.Port), Protocol: corev1.ProtocolTCP, Name: nais.DefaultPortName},
		},
		Resources:       resourceLimits(app.Spec.Resources),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Lifecycle:       lifeCycle(app.Spec.PreStopHookPath),
		Env:             envVars(app),
	}

	if len(app.Spec.Liveness.Path) > 0 {
		c.LivenessProbe = probe(app, app.Spec.Liveness)
	}

	if len(app.Spec.Readiness.Path) > 0 {
		c.ReadinessProbe = probe(app, app.Spec.Readiness)
	}

	return c
}

func leaderElection(app *nais.Application, podSpec *corev1.PodSpec) (spec *corev1.PodSpec) {
	spec = podSpec.DeepCopy()
	spec.Containers = append(spec.Containers, leaderElectionContainer(app.Name, app.Namespace))
	mainContainer := spec.Containers[0].DeepCopy()

	electorPathEnv := corev1.EnvVar{
		Name:  "ELECTOR_PATH",
		Value: "localhost:4040",
	}

	mainContainer.Env = append(mainContainer.Env, electorPathEnv)
	spec.Containers[0] = *mainContainer

	return spec
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

func vaultContainers(app *nais.Application, podSpec *corev1.PodSpec) (*corev1.PodSpec, error) {
	initializer, err := vault.NewInitializer(app)
	if err != nil {
		return nil, fmt.Errorf("while initializing secrets: %s", err)
	}
	spec := initializer.AddVaultContainers(app, podSpec)

	return &spec, nil
}

func defaultEnvVars(app *nais.Application) []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: NaisAppNameEnv, Value: app.ObjectMeta.Name},
		{Name: NaisNamespaceEnv, Value: app.ObjectMeta.Namespace},
		{Name: NaisAppImageEnv, Value: app.Spec.Image},
		{Name: nais.NaisClusterNameEnv, Value: app.Cluster()},
	}
}

// Maps environment variables from ApplicationSpec to the ones we use in PodSpec
func envVars(app *nais.Application) []corev1.EnvVar {
	newEnvVars := defaultEnvVars(app)

	for _, envVar := range app.Spec.Env {
		if envVar.ValueFrom.FieldRef.FieldPath != "" {
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

func podObjectMeta(app *nais.Application) metav1.ObjectMeta {
	objectMeta := app.CreateObjectMeta()

	port := app.Spec.Prometheus.Port
	if len(port) == 0 {
		port = strconv.Itoa(app.Spec.Port)
	}

	objectMeta.Annotations = map[string]string{
		"prometheus.io/scrape": strconv.FormatBool(app.Spec.Prometheus.Enabled),
		"prometheus.io/port":   port,
		"prometheus.io/path":   app.Spec.Prometheus.Path,
	}
	if len(app.Spec.Logformat) > 0 {
		objectMeta.Annotations["nais.io/logformat"] = app.Spec.Logformat
	}

	if len(app.Spec.Logtransform) > 0 {
		objectMeta.Annotations["nais.io/logtransform"] = app.Spec.Logtransform
	}

	return objectMeta
}

func resourceLimits(reqs nais.ResourceRequirements) corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(reqs.Requests.Cpu),
			corev1.ResourceMemory: resource.MustParse(reqs.Requests.Memory),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(reqs.Limits.Cpu),
			corev1.ResourceMemory: resource.MustParse(reqs.Limits.Memory),
		},
	}
}

func lifeCycle(path string) *corev1.Lifecycle {
	if len(path) > 0 {
		return &corev1.Lifecycle{
			PreStop: &corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: path,
					Port: intstr.FromString(nais.DefaultPortName),
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

func probe(app *nais.Application, probe nais.Probe) (k8sprobe *corev1.Probe) {
	port := probe.Port
	if port == 0 {
		port = app.Spec.Port
	}

	k8sprobe = &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: probe.Path,
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

	return
}

func leaderElectionContainer(name, namespace string) corev1.Container {
	return corev1.Container{
		Name:            "elector",
		Image:           "gcr.io/google_containers/leader-elector:0.5",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("100m"),
			},
		},
		Ports: []corev1.ContainerPort{{
			ContainerPort: 4040,
			Protocol:      corev1.ProtocolTCP,
		}},
		Args: []string{fmt.Sprintf("--election=%s", name), "--http=localhost:4040", fmt.Sprintf("--election-namespace=%s", namespace)},
	}
}

func GetContainerByName(containers []corev1.Container, name string) *corev1.Container {
	for i, v := range containers {
		if v.Name == name {
			return &containers[i]
		}
	}

	return nil
}

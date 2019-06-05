package resourcecreator

import (
	"fmt"
	"os"
	"strconv"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"github.com/nais/naiserator/pkg/securelogs"
	"github.com/nais/naiserator/pkg/vault"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// default environment variables
const (
	NaisAppName     = "NAIS_APP_NAME"
	NaisNamespace   = "NAIS_NAMESPACE"
	NaisAppImage    = "NAIS_APP_IMAGE"
	NaisClusterName = "NAIS_CLUSTER_NAME"
)

func Deployment(app *nais.Application, resourceOptions ResourceOptions) (*appsv1.Deployment, error) {
	spec, err := deploymentSpec(app, resourceOptions)
	if err != nil {
		return nil, err
	}
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec:       *spec,
	}, nil
}

func deploymentSpec(app *nais.Application, resourceOptions ResourceOptions) (*appsv1.DeploymentSpec, error) {
	podSpec, err := podSpec(app)
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

func podSpec(app *nais.Application) (*corev1.PodSpec, error) {
	var err error

	podSpec := podSpecBase(app)

	if app.Spec.LeaderElection {
		podSpec = podSpecLeaderElection(app, podSpec)
	}

	podSpec = podSpecCertificateAuthority(podSpec)

	podSpec = podSpecConfigMapFiles(app, podSpec)

	if vault.Enabled() && app.Spec.Vault.Enabled {
		if len(app.Spec.Vault.Mounts) == 0 {
			app.Spec.Vault.Mounts = []nais.SecretPath{
				app.DefaultSecretPath(vault.DefaultKVPath()),
			}
		}
		podSpec, err = podSpecSecrets(app, podSpec)
		if err != nil {
			return nil, err
		}
	}

	if app.Spec.WebProxy {
		podSpec, err = podSpecProxyOpts(podSpec)
		if err != nil {
			return nil, err
		}
	}

	if app.Spec.SecureLogs.Enabled {
		podSpec = podSpecSecureLogs(podSpec)
	}

	return podSpec, err
}

func podSpecConfigMapFiles(app *nais.Application, spec *corev1.PodSpec) *corev1.PodSpec {
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

	if app.Spec.Liveness != (nais.Probe{}) {
		c.LivenessProbe = probe(app.Spec.Liveness)
	}

	if app.Spec.Readiness != (nais.Probe{}) {
		c.ReadinessProbe = probe(app.Spec.Readiness)
	}

	return c
}

func podSpecLeaderElection(app *nais.Application, podSpec *corev1.PodSpec) (spec *corev1.PodSpec) {
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

func podSpecSecureLogs(podSpec *corev1.PodSpec) *corev1.PodSpec {
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

func podSpecSecrets(app *nais.Application, podSpec *corev1.PodSpec) (*corev1.PodSpec, error) {
	initializer, err := vault.NewInitializer(app)
	if err != nil {
		return nil, fmt.Errorf("while initializing secrets: %s", err)
	}
	spec := initializer.AddVaultContainers(podSpec)

	return &spec, nil
}

func defaultEnvVars(app *nais.Application) []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: NaisAppName, Value: app.ObjectMeta.Name},
		{Name: NaisNamespace, Value: app.ObjectMeta.Namespace},
		{Name: NaisAppImage, Value: app.Spec.Image},
		{Name: NaisClusterName, Value: os.Getenv(NaisClusterName)},
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

	objectMeta.Annotations = map[string]string{
		"prometheus.io/scrape": strconv.FormatBool(app.Spec.Prometheus.Enabled),
		"prometheus.io/port":   app.Spec.Prometheus.Port,
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

func probe(probe nais.Probe) (k8sprobe *corev1.Probe) {
	k8sprobe = &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: probe.Path,
				Port: intstr.FromString(nais.DefaultPortName),
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

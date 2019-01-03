package resourcecreator

import (
	"fmt"
	"strconv"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"github.com/nais/naiserator/pkg/vault"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Deployment(app *nais.Application, opts ResourceOptions) (*appsv1.Deployment, error) {
	spec, err := deploymentSpec(app, opts)
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

func deploymentSpec(app *nais.Application, opts ResourceOptions) (*appsv1.DeploymentSpec, error) {
	podspec, err := podSpec(app)
	if err != nil {
		return nil, err
	}
	return &appsv1.DeploymentSpec{
		Replicas: int32p(opts.NumReplicas),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": app.Name},
		},
		Strategy: appsv1.DeploymentStrategy{
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
		},
		ProgressDeadlineSeconds: int32p(300),
		RevisionHistoryLimit:    int32p(10),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: podObjectMeta(app),
			Spec:       *podspec,
		},
	}, nil
}

func podSpec(app *nais.Application) (*corev1.PodSpec, error) {
	var err error

	podSpec := podSpecBase(app)

	if app.Spec.LeaderElection {
		podSpecLeaderElection(app, podSpec)
	}

	podSpec = podSpecCertificateAuthority(podSpec)

	if app.Spec.Vault.Enabled || app.Spec.Secrets {
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

	return podSpec, err
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
		Env:             envVars(app.Spec.Env),
	}

	if app.Spec.Liveness != (nais.Probe{}) {
		c.LivenessProbe = probe(app.Spec.Liveness)
	}

	if app.Spec.Readiness != (nais.Probe{}) {
		c.ReadinessProbe = probe(app.Spec.Readiness)
	}

	return c
}

func podSpecLeaderElection(app *nais.Application, podSpec *corev1.PodSpec) *corev1.PodSpec {
	podSpec.Containers = append(podSpec.Containers, leaderElectionContainer(app.Namespace, app.Namespace))
	mainContainer := &podSpec.Containers[0]

	electorPathEnv := corev1.EnvVar{
		Name:  "ELECTOR_PATH",
		Value: "localhost:4040",
	}

	mainContainer.Env = append(mainContainer.Env, electorPathEnv)

	return podSpec
}

func toSecretPaths(path []nais.SecretPath) []vault.SecretPath {
	vp := make([]vault.SecretPath, len(path))
	for i, p := range path {
		vp[i] = vault.SecretPath{
			KVPath:    p.KvPath,
			MountPath: p.MountPath,
		}
	}
	return vp
}

func podSpecSecrets(app *nais.Application, podSpec *corev1.PodSpec) (*corev1.PodSpec, error) {
	initializer, err := vault.NewInitializer(app.Name, app.Namespace, toSecretPaths(app.Spec.Vault.Mounts))
	if err != nil {
		return nil, fmt.Errorf("while initializing secrets: %s", err)
	}
	spec := initializer.AddInitContainer(podSpec)
	return &spec, nil
}

// Maps environment variables from ApplicationSpec to the ones we use in PodSpec
func envVars(vars []nais.EnvVar) []corev1.EnvVar {
	var newEnvVars []corev1.EnvVar

	for _, envVar := range vars {
		newEnvVars = append(newEnvVars, corev1.EnvVar{Name: envVar.Name, Value: envVar.Value})
	}

	return newEnvVars
}

func podObjectMeta(app *nais.Application) metav1.ObjectMeta {
	objectMeta := app.CreateObjectMeta()

	objectMeta.Annotations = map[string]string{
		"prometheus.io/scrape": strconv.FormatBool(app.Spec.Prometheus.Enabled),
		"prometheus.io/port":   nais.DefaultPortName,
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
    //return &corev1.Lifecycle{}
	return &corev1.Lifecycle{
		PreStop: &corev1.Handler{
			Exec: &corev1.ExecAction{
				Command: []string{"sleep","5"},
			},
		},
	}
}

func probe(probe nais.Probe) *corev1.Probe {
	return &corev1.Probe{
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
}

func leaderElectionContainer(name, ns string) corev1.Container {
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
		Args: []string{"--election=" + name, "--http=localhost:4040", fmt.Sprintf("--election-namespace=%s", ns)},
	}
}

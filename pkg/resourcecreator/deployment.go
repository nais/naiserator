package resourcecreator

import (
	nais "github.com/nais/naiserator/api/types/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strconv"
)

func getDeployment(app *nais.Application) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: getObjectMeta(app),
		Spec:       getDeploymentSpec(app),
	}
}

func getDeploymentSpec(app *nais.Application) appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Replicas: int32p(1),
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
			ObjectMeta: getPodObjectMeta(app),
			Spec:       getPodSpec(app),
		},
	}
}

//TODO mount configmaps, vault initcontainer
func getPodSpec(app *nais.Application) corev1.PodSpec {
	return corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  app.Name,
				Image: app.Spec.Image,
				Ports: []corev1.ContainerPort{
					{ContainerPort: int32(app.Spec.Port), Protocol: corev1.ProtocolTCP, Name: nais.DefaultPortName},
				},
				Resources:       getResourceLimits(app.Spec.Resources),
				LivenessProbe:   getProbe(app.Spec.Healthcheck.Liveness),
				ReadinessProbe:  getProbe(app.Spec.Healthcheck.Readiness),
				ImagePullPolicy: corev1.PullIfNotPresent,
				Lifecycle:       getLifeCycle(app.Spec.PreStopHookPath),
			},
		},
		ServiceAccountName: app.Name,
		RestartPolicy:      corev1.RestartPolicyAlways,
		DNSPolicy:          corev1.DNSClusterFirst,
	}
}

func getPodObjectMeta(app *nais.Application) metav1.ObjectMeta {
	objectMeta := getObjectMeta(app)

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

func getResourceLimits(reqs nais.ResourceRequirements) corev1.ResourceRequirements {
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

func getLifeCycle(path string) *corev1.Lifecycle {
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

	return &corev1.Lifecycle{}
}

func getProbe(probe nais.Probe) *corev1.Probe {
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

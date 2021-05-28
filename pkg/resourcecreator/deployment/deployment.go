package deployment

import (
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"

	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Create(objectMeta metav1.ObjectMeta, resourceOptions resource.Options, operations *resource.Operations, naisAnnotations map[string]string, naisStrategy nais_io_v1alpha1.Strategy, image, preStopHookPath, logFormat, logTransform string, appPort int, naisResources nais_io_v1alpha1.ResourceRequirements, livenessProbe, readinessProve, startupProbe *nais_io_v1alpha1.Probe, naisFilesFrom []nais_io_v1alpha1.FilesFrom, naisEnvFrom []nais_io_v1alpha1.EnvFrom, naisEnvVars []nais_io_v1alpha1.EnvVar, prometheusConfig *nais_io_v1alpha1.PrometheusConfig) (*appsv1.Deployment, error) {
	spec, err := deploymentSpec(objectMeta, resourceOptions, naisStrategy, image, preStopHookPath, logFormat, logTransform, appPort, naisResources, livenessProbe, readinessProve, startupProbe, naisFilesFrom, naisEnvFrom, naisEnvVars, prometheusConfig)
	if err != nil {
		return nil, err
	}

	if val, ok := naisAnnotations["kubernetes.io/change-cause"]; ok {
		if objectMeta.Annotations == nil {
			objectMeta.Annotations = make(map[string]string)
		}

		objectMeta.Annotations["kubernetes.io/change-cause"] = val
	}

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: objectMeta,
		Spec:       *spec,
	}

	*operations = append(*operations, resource.Operation{Resource: deployment, Operation: resource.OperationCreateOrUpdate})
	return deployment, nil
}

func deploymentSpec(objectMeta metav1.ObjectMeta, resourceOptions resource.Options, naisStrategy nais_io_v1alpha1.Strategy, image, preStopHookPath, logFormat, logTransform string, appPort int, naisResources nais_io_v1alpha1.ResourceRequirements, livenessProbe, readinessProve, startupProbe *nais_io_v1alpha1.Probe, naisFilesFrom []nais_io_v1alpha1.FilesFrom, naisEnvFrom []nais_io_v1alpha1.EnvFrom, naisEnvVars []nais_io_v1alpha1.EnvVar, prometheusConfig *nais_io_v1alpha1.PrometheusConfig) (*appsv1.DeploymentSpec, error) {
	podSpec, err := pod.CreateSpec(objectMeta, resourceOptions, image, preStopHookPath, appPort, naisResources, livenessProbe, readinessProve, startupProbe, naisFilesFrom, naisEnvFrom, naisEnvVars)
	if err != nil {
		return nil, err
	}

	var strategy appsv1.DeploymentStrategy

	if naisStrategy.Type == nais_io_v1alpha1.DeploymentStrategyRecreate {
		strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		}
	} else if naisStrategy.Type == nais_io_v1alpha1.DeploymentStrategyRollingUpdate {
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
		Replicas: util.Int32p(resourceOptions.NumReplicas),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": objectMeta.Name},
		},
		Strategy:                strategy,
		ProgressDeadlineSeconds: util.Int32p(300),
		RevisionHistoryLimit:    util.Int32p(10),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: pod.ObjectMeta(objectMeta, appPort, prometheusConfig, logFormat, logTransform),
			Spec:       *podSpec,
		},
	}, nil
}


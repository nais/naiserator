package deployment

import (
	"fmt"

	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Create(app *nais_io_v1alpha1.Application, ast *resource.Ast, resourceOptions resource.Options) error {
	objectMeta := resource.CreateObjectMeta(app)
	spec, err := deploymentSpec(app, ast, resourceOptions)
	if err != nil {
		return fmt.Errorf("create deployment: %w", err)
	}

	if val, ok := app.Annotations["kubernetes.io/change-cause"]; ok {
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

	ast.AppendOperation(resource.OperationCreateOrUpdate, deployment)

	return nil
}

func deploymentSpec(app *nais_io_v1alpha1.Application, ast *resource.Ast, resourceOptions resource.Options) (*appsv1.DeploymentSpec, error) {
	podSpec, err := pod.CreateSpec(ast, resourceOptions, app.Name, corev1.RestartPolicyAlways)
	if err != nil {
		return nil, err
	}

	var strategy appsv1.DeploymentStrategy

	if app.Spec.Strategy.Type == nais_io_v1alpha1.DeploymentStrategyRecreate {
		strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		}
	} else if app.Spec.Strategy.Type == nais_io_v1alpha1.DeploymentStrategyRollingUpdate {
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
			MatchLabels: map[string]string{"app": app.Name},
		},
		Strategy:                strategy,
		ProgressDeadlineSeconds: util.Int32p(300),
		RevisionHistoryLimit:    util.Int32p(10),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: pod.CreateAppObjectMeta(app, ast),
			Spec:       *podSpec,
		},
	}, nil
}

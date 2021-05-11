package deployment

import (
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Create(app *nais.Application, resourceOptions resource.Options, operations *resource.Operations) (*appsv1.Deployment, error) {
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

func deploymentSpec(app *nais.Application, resourceOptions resource.Options) (*appsv1.DeploymentSpec, error) {
	podSpec, err := pod.Spec(resourceOptions, app)
	if err != nil {
		return nil, err
	}

	var strategy appsv1.DeploymentStrategy

	if app.Spec.Strategy == nil {
		log.Error("BUG: strategy is nil; should be fixed by NilFix")
		app.Spec.Strategy = &nais.Strategy{Type: nais.DeploymentStrategyRollingUpdate}
	}

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
		Replicas: util.Int32p(resourceOptions.NumReplicas),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": app.Name},
		},
		Strategy:                strategy,
		ProgressDeadlineSeconds: util.Int32p(300),
		RevisionHistoryLimit:    util.Int32p(10),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: pod.ObjectMeta(app),
			Spec:       *podSpec,
		},
	}, nil
}


package deployment

import (
	"fmt"
	"strings"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type Source interface {
	resource.Source
	GetCleanup() *nais_io_v1.Cleanup
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
	GetReplicas() *nais_io_v1.Replicas
	GetResources() *nais_io_v1.ResourceRequirements
	GetStartup() *nais_io_v1.Probe
	GetStrategy() *nais_io_v1.Strategy
}

func Create(app Source, ast *resource.Ast, resourceOptions resource.Options) error {
	objectMeta := resource.CreateObjectMeta(app)
	spec, err := deploymentSpec(app, ast, resourceOptions)
	if err != nil {
		return fmt.Errorf("create deployment: %w", err)
	}

	if val, ok := app.GetAnnotations()["kubernetes.io/change-cause"]; ok {
		objectMeta.Annotations["kubernetes.io/change-cause"] = val
	}

	objectMeta = addCleanupLabels(app, objectMeta)
	objectMeta.Annotations["reloader.stakater.com/search"] = "true"

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

func addCleanupLabels(app Source, meta metav1.ObjectMeta) metav1.ObjectMeta {
	cleanup := app.GetCleanup()

	if cleanup == nil {
		meta.Annotations["babylon.nais.io/enabled"] = "true"
		meta.Annotations["babylon.nais.io/strategy"] = "abort-rollout,downscale"
		meta.Annotations["babylon.nais.io/grace-period"] = "24h"

		return meta
	}

	meta.Annotations["babylon.nais.io/enabled"] = fmt.Sprintf("%t", cleanup.Enabled)
	var strategies []string
	for _, s := range cleanup.Strategy {
		strategies = append(strategies, string(s))
	}
	meta.Annotations["babylon.nais.io/strategy"] = strings.Join(strategies, ",")
	meta.Annotations["babylon.nais.io/grace-period"] = cleanup.GracePeriod

	return meta
}

func deploymentSpec(app Source, ast *resource.Ast, resourceOptions resource.Options) (*appsv1.DeploymentSpec, error) {
	podSpec, err := pod.CreateSpec(ast, resourceOptions, app.GetName(), app.GetAnnotations(), corev1.RestartPolicyAlways)
	if err != nil {
		return nil, err
	}

	var strategy appsv1.DeploymentStrategy

	if app.GetStrategy().Type == nais_io_v1alpha1.DeploymentStrategyRecreate {
		strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		}
	} else if app.GetStrategy().Type == nais_io_v1alpha1.DeploymentStrategyRollingUpdate {
		strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDeployment{
				MaxUnavailable: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: int32(0),
				},
				MaxSurge: &intstr.IntOrString{
					Type:   intstr.String,
					StrVal: "25%",
				},
			},
		}
	}

	return &appsv1.DeploymentSpec{
		Replicas: util.Int32p(int32(*app.GetReplicas().Min)),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": app.GetName()},
		},
		Strategy:                strategy,
		ProgressDeadlineSeconds: util.Int32p(300),
		RevisionHistoryLimit:    util.Int32p(10),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: pod.CreateAppObjectMeta(app, ast, &resourceOptions),
			Spec:       *podSpec,
		},
	}, nil
}

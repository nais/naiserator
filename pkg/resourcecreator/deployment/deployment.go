package deployment

import (
	"fmt"
	"time"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
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
	GetReplicas() *nais_io_v1.Replicas
	GetResources() *nais_io_v1.ResourceRequirements
	GetStartup() *nais_io_v1.Probe
	GetStrategy() *nais_io_v1.Strategy
	GetTerminationGracePeriodSeconds() *int64
	GetTTL() string
}

type Config interface {
	pod.Config
	GetNumReplicas() int32
}

func Create(app Source, ast *resource.Ast, cfg Config) error {
	objectMeta := resource.CreateObjectMeta(app)
	spec, err := deploymentSpec(app, ast, cfg)
	if err != nil {
		return fmt.Errorf("NAISERATOR-4662: create deployment: %w", err)
	}

	if val, ok := app.GetAnnotations()["kubernetes.io/change-cause"]; ok {
		objectMeta.Annotations["kubernetes.io/change-cause"] = val
	}

	objectMeta.Annotations["reloader.stakater.com/search"] = "true"

	if app.GetTTL() != "" {
		d, err := time.ParseDuration(app.GetTTL())
		if err != nil {
			return fmt.Errorf("NAISERATOR-4232: parsing TTL: %w", err)
		}

		objectMeta.Annotations["euthanaisa.nais.io/kill-after"] = time.Now().Add(d).Format(time.RFC3339)
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

func deploymentSpec(app Source, ast *resource.Ast, cfg Config) (*appsv1.DeploymentSpec, error) {
	podSpec, err := pod.CreateSpec(ast, cfg, app.GetName(), app.GetAnnotations(), corev1.RestartPolicyAlways, app.GetTerminationGracePeriodSeconds())
	if err != nil {
		return nil, err
	}

	return &appsv1.DeploymentSpec{
		Replicas: util.Int32p(cfg.GetNumReplicas()),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": app.GetName()},
		},
		Strategy:                deploymentStrategy(app),
		ProgressDeadlineSeconds: util.Int32p(300),
		RevisionHistoryLimit:    util.Int32p(3),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: pod.CreateAppObjectMeta(app, ast, cfg),
			Spec:       *podSpec,
		},
	}, nil
}

func deploymentStrategy(app Source) appsv1.DeploymentStrategy {
	if app.GetStrategy().Type == nais_io_v1alpha1.DeploymentStrategyRecreate {
		return appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		}
	}

	rollingUpdateConfig := &appsv1.RollingUpdateDeployment{
		MaxUnavailable: &intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: int32(0),
		},
		MaxSurge: &intstr.IntOrString{
			Type:   intstr.String,
			StrVal: "25%",
		},
	}

	override := app.GetStrategy().RollingUpdate
	if override != nil {
		if override.MaxSurge != nil {
			rollingUpdateConfig.MaxSurge = override.MaxSurge
		}

		if override.MaxUnavailable != nil {
			rollingUpdateConfig.MaxUnavailable = override.MaxUnavailable
		}
	}

	return appsv1.DeploymentStrategy{
		Type:          appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: rollingUpdateConfig,
	}
}

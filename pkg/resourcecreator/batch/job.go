package batch

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

type Config interface {
	pod.Config
}

func CreateJobSpec(naisjob *nais_io_v1.Naisjob, ast *resource.Ast, cfg Config) (batchv1.JobSpec, error) {

	podSpec, err := pod.CreateSpec(ast, cfg, naisjob.GetName(), naisjob.Annotations, RestartPolicy(naisjob.Spec.RestartPolicy), naisjob.Spec.TerminationGracePeriodSeconds)
	if err != nil {
		return batchv1.JobSpec{}, err
	}

	jobSpec := batchv1.JobSpec{
		ActiveDeadlineSeconds: naisjob.Spec.ActiveDeadlineSeconds,
		BackoffLimit:          naisjob.Spec.BackoffLimit,
		Completions:           naisjob.Spec.Completions,
		Parallelism:           naisjob.Spec.Parallelism,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: pod.CreateNaisjobObjectMeta(naisjob, ast, cfg),
			Spec:       *podSpec,
		},
		TTLSecondsAfterFinished: naisjob.Spec.TTLSecondsAfterFinished,
	}

	return jobSpec, nil
}

func CreateJob(naisjob *nais_io_v1.Naisjob, ast *resource.Ast, cfg Config) error {

	objectMeta := resource.CreateObjectMeta(naisjob)

	if val, ok := naisjob.GetAnnotations()["kubernetes.io/change-cause"]; ok {
		objectMeta.Annotations["kubernetes.io/change-cause"] = val
	}

	jobSpec, err := CreateJobSpec(naisjob, ast, cfg)
	if err != nil {
		return err
	}

	job := batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: objectMeta,
		Spec:       jobSpec,
	}

	ast.AppendOperation(resource.OperationCreateOrRecreate, &job)
	return nil
}

func DeleteJob(naisjob *nais_io_v1.Naisjob, ast *resource.Ast) error {
	objectMeta := resource.CreateObjectMeta(naisjob)

	job := batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: objectMeta,
	}

	ast.AppendOperation(resource.OperationDeleteIfExists, &job)

	return nil
}

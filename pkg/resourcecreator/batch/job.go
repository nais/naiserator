package batch

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateJobSpec(naisjob *nais_io_v1.Naisjob, ast *resource.Ast, resourceOptions resource.Options) (batchv1.JobSpec, error) {
	podSpec, err := pod.CreateSpec(ast, resourceOptions, naisjob.GetName(), corev1.RestartPolicyNever)
	if err != nil {
		return batchv1.JobSpec{}, err
	}

	jobSpec := batchv1.JobSpec{
		ActiveDeadlineSeconds: naisjob.Spec.ActiveDeadlineSeconds,
		BackoffLimit:          util.Int32p(naisjob.Spec.BackoffLimit),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: pod.CreateNaisjobObjectMeta(naisjob, ast),
			Spec:       *podSpec,
		},
		TTLSecondsAfterFinished: naisjob.Spec.TTLSecondsAfterFinished,
	}

	return jobSpec, nil
}

func CreateJob(naisjob *nais_io_v1.Naisjob, ast *resource.Ast, resourceOptions resource.Options) error {

	objectMeta := resource.CreateObjectMeta(naisjob)

	if val, ok := naisjob.GetAnnotations()["kubernetes.io/change-cause"]; ok {
		objectMeta.Annotations["kubernetes.io/change-cause"] = val
	}

	jobSpec, err := CreateJobSpec(naisjob, ast, resourceOptions)
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

	ast.AppendOperation(resource.OperationCreateOrUpdate, &job)
	return nil
}

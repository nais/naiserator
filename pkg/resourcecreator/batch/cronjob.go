package batch

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	"k8s.io/api/batch/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateCronJob(naisjob *nais_io_v1.Naisjob, ast *resource.Ast, resourceOptions resource.Options) error {

	objectMeta := resource.CreateObjectMeta(naisjob)

	if val, ok := naisjob.GetAnnotations()["kubernetes.io/change-cause"]; ok {
		objectMeta.Annotations["kubernetes.io/change-cause"] = val
	}

	jobSpec, err := CreateJobSpec(naisjob, ast, resourceOptions)
	if err != nil {
		return err
	}

	cronJob := v1beta1.CronJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CronJob",
			APIVersion: "batch/v1beta1",
		},
		ObjectMeta: objectMeta,
		Spec: v1beta1.CronJobSpec{
			Schedule: naisjob.Spec.Schedule,
			JobTemplate: v1beta1.JobTemplateSpec{
				ObjectMeta: resource.CreateObjectMeta(naisjob),
				Spec:       jobSpec,
			},
			SuccessfulJobsHistoryLimit: util.Int32p(naisjob.Spec.SuccessfulJobsHistoryLimit),
			FailedJobsHistoryLimit:     util.Int32p(naisjob.Spec.FailedJobsHistoryLimit),
			ConcurrencyPolicy:          v1beta1.ConcurrencyPolicy(naisjob.GetConcurrencyPolicy()),
		},
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, &cronJob)
	return nil
}

package batch

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	"k8s.io/api/batch/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateCronJob(naisjob *nais_io_v1.Naisjob, ast *resource.Ast) {

	objectMeta := naisjob.CreateObjectMeta()

	if val, ok := naisjob.GetAnnotations()["kubernetes.io/change-cause"]; ok {
		if objectMeta.Annotations == nil {
			objectMeta.Annotations = make(map[string]string)
		}

		objectMeta.Annotations["kubernetes.io/change-cause"] = val
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
				ObjectMeta: naisjob.CreateObjectMeta(),
				Spec:       ast.JobSpec,
			},
			SuccessfulJobsHistoryLimit: util.Int32p(naisjob.Spec.SuccessfulJobsHistoryLimit),
			FailedJobsHistoryLimit:     util.Int32p(naisjob.Spec.FailedJobsHistoryLimit),
		},
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, &cronJob)
}

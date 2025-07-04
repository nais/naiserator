package batch

import (
	"fmt"
	"time"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
)

func CreateCronJob(naisjob *nais_io_v1.Naisjob, ast *resource.Ast, cfg Config) error {
	objectMeta := resource.CreateObjectMeta(naisjob)

	if val, ok := naisjob.GetAnnotations()["kubernetes.io/change-cause"]; ok {
		objectMeta.Annotations["kubernetes.io/change-cause"] = val
	}

	jobSpec, err := CreateJobSpec(naisjob, ast, cfg)
	if err != nil {
		return err
	}

	if naisjob.Spec.TTL != "" {
		d, err := time.ParseDuration(naisjob.Spec.TTL)
		if err != nil {
			return fmt.Errorf("parsing TTL: %w", err)
		}

		objectMeta.Annotations["euthanaisa.nais.io/kill-after"] = time.Now().Add(d).Format(time.RFC3339)
		objectMeta.Labels["euthanaisa.nais.io/enabled"] = "true"
	}

	cronJob := batchv1.CronJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CronJob",
			APIVersion: "batch/v1",
		},
		ObjectMeta: objectMeta,
		Spec: batchv1.CronJobSpec{
			TimeZone: naisjob.Spec.TimeZone,
			Schedule: naisjob.Spec.Schedule,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: resource.CreateObjectMeta(naisjob),
				Spec:       jobSpec,
			},
			SuccessfulJobsHistoryLimit: util.Int32p(naisjob.Spec.SuccessfulJobsHistoryLimit),
			FailedJobsHistoryLimit:     util.Int32p(naisjob.Spec.FailedJobsHistoryLimit),
			ConcurrencyPolicy:          batchv1.ConcurrencyPolicy(naisjob.GetConcurrencyPolicy()),
		},
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, &cronJob)
	return nil
}

func DeleteCronJob(naisjob *nais_io_v1.Naisjob, ast *resource.Ast) error {
	objectMeta := resource.CreateObjectMeta(naisjob)

	cronJob := batchv1.CronJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CronJob",
			APIVersion: "batch/v1",
		},
		ObjectMeta: objectMeta,
	}

	ast.AppendOperation(resource.OperationDeleteIfExists, &cronJob)

	return nil
}

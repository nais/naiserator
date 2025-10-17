package batch

import (
	"fmt"
	"time"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// All Naisjob are CronJobs, if no schedule is set we run it once on creation and set suspend to true. The job can be rerun on demand.
// see syncronizer/monitoring.go monitorNaisjob

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
	schedule := naisjob.Spec.Schedule
	suspend := false
	if schedule == "" {
		schedule = "0 0 1 1 *"
		suspend = true
	}

	cronJob := batchv1.CronJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CronJob",
			APIVersion: "batch/v1",
		},
		ObjectMeta: objectMeta,
		Spec: batchv1.CronJobSpec{
			TimeZone: naisjob.Spec.TimeZone,
			Schedule: schedule,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: resource.CreateObjectMeta(naisjob),
				Spec:       jobSpec,
			},
			Suspend:                    &suspend,
			SuccessfulJobsHistoryLimit: util.Int32p(naisjob.Spec.SuccessfulJobsHistoryLimit),
			FailedJobsHistoryLimit:     util.Int32p(naisjob.Spec.FailedJobsHistoryLimit),
			ConcurrencyPolicy:          batchv1.ConcurrencyPolicy(naisjob.GetConcurrencyPolicy()),
		},
	}
	ast.AppendOperation(resource.OperationCreateOrUpdate, &cronJob)
	return nil
}

func CreateJobFromCronJob(cronJob *batchv1.CronJob) (*batchv1.Job, error) {
	return &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cronJob.Name,
			Namespace: cronJob.GetNamespace(),
			Labels:    cronJob.GetLabels(),
			Annotations: map[string]string{
				"cronjob.kubernetes.io/instantiate": "manual",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         cronJob.APIVersion,
					Kind:               cronJob.Kind,
					Name:               cronJob.GetName(),
					UID:                cronJob.GetUID(),
					Controller:         ptr.To(true),
					BlockOwnerDeletion: ptr.To(true),
				},
			},
		},
		Spec: cronJob.Spec.JobTemplate.Spec,
	}, nil
}

package batch

import (
	"fmt"

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
			FailedJobsHistoryLimit:     naisjob.Spec.FailedJobsHistoryLimit,
			ConcurrencyPolicy:          batchv1.ConcurrencyPolicy(naisjob.GetConcurrencyPolicy()),
		},
	}
	ast.AppendOperation(resource.OperationCreateOrUpdate, &cronJob)
	return nil
}

func truncateString(str string, max int) string {
	truncated := ""
	count := 0
	if len(str) < max {
		return str
	}

	for _, char := range str {
		truncated += string(char)
		count++
		if count >= max {
			break
		}
	}
	return truncated
}

func CreateJobFromCronJob(cronJob *batchv1.CronJob) (*batchv1.Job, error) {
	return &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", truncateString(cronJob.Name, 48), cronJob.Generation),
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

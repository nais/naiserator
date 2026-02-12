package batch

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

type Config interface {
	pod.Config
}

func CreateJobSpec(naisjob *nais_io_v1.Naisjob, ast *resource.Ast, cfg Config) (batchv1.JobSpec, error) {
	podSpec, err := pod.CreateSpec(ast, cfg, naisjob.GetName(), naisjob.Annotations, RestartPolicy(naisjob.Spec.RestartPolicy), naisjob.Spec.TerminationGracePeriodSeconds)
	if err != nil {
		return batchv1.JobSpec{}, err
	}

	var completionMode *batchv1.CompletionMode
	if naisjob.Spec.CompletionMode != nil {
		completionMode = new(batchv1.CompletionMode(*naisjob.Spec.CompletionMode))
	}

	jobSpec := batchv1.JobSpec{
		ActiveDeadlineSeconds: naisjob.Spec.ActiveDeadlineSeconds,
		BackoffLimit:          naisjob.Spec.BackoffLimit,
		Completions:           naisjob.Spec.Completions,
		CompletionMode:        completionMode,
		Parallelism:           naisjob.Spec.Parallelism,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: pod.CreateNaisjobObjectMeta(naisjob, ast, cfg),
			Spec:       *podSpec,
		},
		TTLSecondsAfterFinished: naisjob.Spec.TTLSecondsAfterFinished,
	}

	return jobSpec, nil
}

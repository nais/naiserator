package batch

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateJobSpec(naisjob *nais_io_v1.Naisjob, ast *resource.Ast, resourceOptions resource.Options) error {

	podSpec, err := pod.CreateSpec(ast, resourceOptions, naisjob.GetName())
	if err != nil {
		return err
	}

	jobSpec := v1.JobSpec{
		ActiveDeadlineSeconds: util.Int64p(naisjob.Spec.ActiveDeadlineSeconds),
		BackoffLimit:          util.Int32p(naisjob.Spec.BackoffLimit),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": naisjob.GetName()},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: pod.CreateNaisjobObjectMeta(naisjob, ast),
			Spec:       *podSpec,
		},
		TTLSecondsAfterFinished: util.Int32p(naisjob.Spec.TTLSecondsAfterFinished),
	}

	ast.JobSpec = jobSpec
	return nil
}

func CreateJob(source resource.Source, ast *resource.Ast) {

	objectMeta := source.CreateObjectMeta()

	if val, ok := source.GetAnnotations()["kubernetes.io/change-cause"]; ok {
		if objectMeta.Annotations == nil {
			objectMeta.Annotations = make(map[string]string)
		}

		objectMeta.Annotations["kubernetes.io/change-cause"] = val
	}

	job := v1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: objectMeta,
		Spec:       ast.JobSpec,
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, &job)
}

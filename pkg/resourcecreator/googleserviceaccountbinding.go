package resourcecreator

import (
	"fmt"

	google_iam_crd "github.com/nais/naiserator/pkg/apis/iam.cnrm.cloud.google.com/v1alpha1"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GoogleServiceAccountBinding(app *nais.Application, sa *google_iam_crd.IAMServiceAccount, projectId string) google_iam_crd.IAMPolicy {
	member := fmt.Sprintf("serviceAccount:%s.svc.id.goog[%s/%s]",projectId,app.Namespace,app.Name)
	objectMeta := app.CreateObjectMeta()
	objectMeta.Namespace = GoogleIAMServiceAccountNamespace
	objectMeta.Name = app.CreateAppNamespaceHash()

	return google_iam_crd.IAMPolicy{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "IAMPolicy",
			APIVersion: GoogleIAMAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec: google_iam_crd.IAMPolicySpec{
			ResourceRef: &google_iam_crd.ResourceRef{
				ApiVersion: GoogleIAMAPIVersion,
				Kind:       "IAMServiceAccount",
				Name:       sa.Name,
			},
			Bindings: []google_iam_crd.Bindings{
				{
					Role:    "roles/iam.workloadIdentityUser",
					Members: []string{member},
				},
			},
		},
	}
}

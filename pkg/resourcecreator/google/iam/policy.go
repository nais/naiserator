package google_iam

import (
	"fmt"

	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/util"

	google_iam_crd "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Policy(app *nais.Application, sa *google_iam_crd.IAMServiceAccount, projectId string) google_iam_crd.IAMPolicy {
	member := fmt.Sprintf("serviceAccount:%s.svc.id.goog[%s/%s]", projectId, app.Namespace, app.Name)
	objectMeta := app.CreateObjectMeta()
	objectMeta.Namespace = google.IAMServiceAccountNamespace
	objectMeta.Name = app.CreateAppNamespaceHash()
	iamPolicy := google_iam_crd.IAMPolicy{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "IAMPolicy",
			APIVersion: google.IAMAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec: google_iam_crd.IAMPolicySpec{
			ResourceRef: &google_iam_crd.ResourceRef{
				ApiVersion: google.IAMAPIVersion,
				Kind:       "IAMServiceAccount",
				Name:       &sa.Name,
			},
			Bindings: []google_iam_crd.Bindings{
				{
					Role:    "roles/iam.workloadIdentityUser",
					Members: []string{member},
				},
			},
		},
	}

	util.SetAnnotation(&iamPolicy, google.ProjectIdAnnotation, projectId)

	return iamPolicy
}

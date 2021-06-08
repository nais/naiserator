package google_iam

import (
	"fmt"

	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"

	google_iam_crd "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreatePolicy(source resource.Source, sa *google_iam_crd.IAMServiceAccount, projectId string) google_iam_crd.IAMPolicy {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = resource.CreateAppNamespaceHash(source)
	objectMeta.Namespace = google.IAMServiceAccountNamespace
	member := fmt.Sprintf("serviceAccount:%s.svc.id.goog[%s/%s]", projectId, source.GetNamespace(), source.GetName())
	iamPolicy := google_iam_crd.IAMPolicy{
		TypeMeta: metav1.TypeMeta{
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

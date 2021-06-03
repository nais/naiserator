package google_iam

import (
	google_iam_crd "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateServiceAccount(source resource.Source, projectId string) google_iam_crd.IAMServiceAccount {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = resource.CreateAppNamespaceHash(source)
	objectMeta.Namespace = google.IAMServiceAccountNamespace
	objectMeta.Annotations["nais.io/team"] = objectMeta.Namespace

	iamServiceAccount := google_iam_crd.IAMServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IAMServiceAccount",
			APIVersion: google.IAMAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec: google_iam_crd.IAMServiceAccountSpec{
			DisplayName: source.GetName(),
		},
	}

	util.SetAnnotation(&iamServiceAccount, "nais.io/team", source.GetNamespace())
	util.SetAnnotation(&iamServiceAccount, google.ProjectIdAnnotation, projectId)

	return iamServiceAccount
}

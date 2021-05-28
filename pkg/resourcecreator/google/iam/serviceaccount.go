package google_iam

import (
	google_iam_crd "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateServiceAccount(objectMeta metav1.ObjectMeta, projectId, appNamespaceHash string) google_iam_crd.IAMServiceAccount {
	appName := objectMeta.Name
	appNamespace := objectMeta.Namespace
	objectMeta.Annotations["nais.io/team"] = objectMeta.Namespace
	objectMeta.Namespace = google.IAMServiceAccountNamespace
	objectMeta.Name = appNamespaceHash

	iamServiceAccount := google_iam_crd.IAMServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IAMServiceAccount",
			APIVersion: google.IAMAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec: google_iam_crd.IAMServiceAccountSpec{
			DisplayName: appName,
		},
	}

	util.SetAnnotation(&iamServiceAccount, "nais.io/team", appNamespace)
	util.SetAnnotation(&iamServiceAccount, google.ProjectIdAnnotation, projectId)

	return iamServiceAccount
}

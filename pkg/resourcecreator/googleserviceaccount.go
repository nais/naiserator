package resourcecreator

import (
	google_iam_crd "github.com/nais/naiserator/pkg/apis/iam.cnrm.cloud.google.com/v1alpha1"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GoogleServiceAccount(app *nais.Application) google_iam_crd.IAMServiceAccount {
	objectMeta := app.CreateObjectMeta()
	objectMeta.Annotations["nais.io/team"] = app.Namespace
	objectMeta.Namespace = GoogleIAMServiceAccountNamespace
	objectMeta.Name = app.CreateAppNamespaceHash()

	return google_iam_crd.IAMServiceAccount{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "IAMServiceAccount",
			APIVersion: GoogleIAMAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec: google_iam_crd.IAMServiceAccountSpec{
			DisplayName: app.Name,
		},
	}
}

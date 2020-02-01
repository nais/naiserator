package resourcecreator

import (
	google_iam_crd "github.com/nais/naiserator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GoogleIAMServiceAccount(app *nais.Application, projectId  string) google_iam_crd.IAMServiceAccount {
	objectMeta := app.CreateObjectMeta()
	objectMeta.Annotations["nais.io/team"] = app.Namespace
	objectMeta.Namespace = GoogleIAMServiceAccountNamespace
	objectMeta.Name = app.CreateAppNamespaceHash()

	iamServiceAccount := google_iam_crd.IAMServiceAccount{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "IAMServiceAccount",
			APIVersion: GoogleIAMAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec: google_iam_crd.IAMServiceAccountSpec{
			DisplayName: app.Name,
		},
	}

	setAnnotation(&iamServiceAccount, "nais.io/team", app.Namespace)
	setAnnotation(&iamServiceAccount, GoogleProjectIdAnnotation, projectId)

	return iamServiceAccount
}

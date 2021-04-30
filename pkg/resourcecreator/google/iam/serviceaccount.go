package google_iam

import (
	google_iam_crd "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/util"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GoogleIAMServiceAccount(app *nais.Application, projectId string) google_iam_crd.IAMServiceAccount {
	objectMeta := app.CreateObjectMeta()
	objectMeta.Annotations["nais.io/team"] = app.Namespace
	objectMeta.Namespace = google.GoogleIAMServiceAccountNamespace
	objectMeta.Name = app.CreateAppNamespaceHash()

	iamServiceAccount := google_iam_crd.IAMServiceAccount{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "IAMServiceAccount",
			APIVersion: google.GoogleIAMAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec: google_iam_crd.IAMServiceAccountSpec{
			DisplayName: app.Name,
		},
	}

	util.SetAnnotation(&iamServiceAccount, "nais.io/team", app.Namespace)
	util.SetAnnotation(&iamServiceAccount, google.GoogleProjectIdAnnotation, projectId)

	return iamServiceAccount
}

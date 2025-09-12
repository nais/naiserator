package postgres

import (
	"fmt"

	"github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func createIAMPolicy(source Source, ast *resource.Ast, projectId, pgNamespace string) {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = fmt.Sprintf("pg-%s", source.GetNamespace())
	objectMeta.Namespace = google.IAMServiceAccountNamespace
	objectMeta.OwnerReferences = nil
	delete(objectMeta.Labels, "app")

	iamPolicy := iam_cnrm_cloud_google_com_v1beta1.IAMPolicy{
		TypeMeta: v1.TypeMeta{
			Kind:       "IAMPolicy",
			APIVersion: google.IAMAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec: iam_cnrm_cloud_google_com_v1beta1.IAMPolicySpec{
			ResourceRef: &iam_cnrm_cloud_google_com_v1beta1.ResourceRef{
				ApiVersion: google.IAMAPIVersion,
				Kind:       "IAMServiceAccount",
				Name:       ptr.To("postgres-pod"),
			},
			Bindings: []iam_cnrm_cloud_google_com_v1beta1.Bindings{
				{
					Role: "roles/iam.workloadIdentityUser",
					Members: []string{
						fmt.Sprintf("serviceAccount:%s.svc.id.goog[%s/postgres-pod]", projectId, pgNamespace),
					},
				},
			},
		},
	}

	util.SetAnnotation(&iamPolicy, google.ProjectIdAnnotation, projectId)

	ast.AppendOperation(resource.OperationCreateIfNotExists, &iamPolicy)
}

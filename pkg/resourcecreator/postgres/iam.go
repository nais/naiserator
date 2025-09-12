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

func createIAMPolicyMember(source Source, ast *resource.Ast, projectId, pgNamespace string) {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = "postgres-pod"
	objectMeta.OwnerReferences = nil
	delete(objectMeta.Labels, "app")

	iamPolicyMember := iam_cnrm_cloud_google_com_v1beta1.IAMPolicyMember{
		TypeMeta: v1.TypeMeta{
			Kind:       "IAMPolicyMember",
			APIVersion: google.IAMAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec: iam_cnrm_cloud_google_com_v1beta1.IAMPolicyMemberSpec{
			Member: fmt.Sprintf("serviceAccount:%s.svc.id.goog[%s/postgres-pod]", projectId, pgNamespace),
			Role:   "roles/iam.workloadIdentityUser",
			ResourceRef: iam_cnrm_cloud_google_com_v1beta1.ResourceRef{
				ApiVersion: google.IAMAPIVersion,
				Kind:       "IAMServiceAccount",
				Name:       ptr.To("postgres-pod"),
			},
		},
	}

	util.SetAnnotation(&iamPolicyMember, google.ProjectIdAnnotation, projectId)

	ast.AppendOperation(resource.OperationCreateIfNotExists, &iamPolicyMember)
}

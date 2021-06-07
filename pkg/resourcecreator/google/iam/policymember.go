package google_iam

import (
	"fmt"
	"strings"

	"github.com/mitchellh/hashstructure"
	google_iam_crd "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func policyMember(source resource.Source, policy nais.CloudIAMPermission, googleProjectId, googleTeamProjectId string) (*google_iam_crd.IAMPolicyMember, error) {
	name, err := createName(source.GetName(), policy)
	if err != nil {
		return nil, err
	}
	externalName := formatExternalName(googleTeamProjectId, policy.Resource.Name)
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = name
	policyMember := &google_iam_crd.IAMPolicyMember{
		ObjectMeta: objectMeta,
		TypeMeta: metav1.TypeMeta{
			Kind:       "IAMPolicyMember",
			APIVersion: google.IAMAPIVersion,
		},
		Spec: google_iam_crd.IAMPolicyMemberSpec{
			Member: fmt.Sprintf("serviceAccount:%s", google.GcpServiceAccountName(resource.CreateAppNamespaceHash(source), googleProjectId)),
			Role:   policy.Role,
			ResourceRef: google_iam_crd.ResourceRef{
				ApiVersion: policy.Resource.APIVersion,
				External:   &externalName,
				Kind:       policy.Resource.Kind,
			},
		},
	}

	util.SetAnnotation(policyMember, google.ProjectIdAnnotation, googleTeamProjectId)

	return policyMember, nil
}

func createName(appName string, policy nais.CloudIAMPermission) (string, error) {
	hash, err := hashstructure.Hash(policy, nil)
	if err != nil {
		return "", fmt.Errorf("while calculating hash from policy: %w", err)
	}
	basename := fmt.Sprintf("%s-%s-%x", appName, strings.ToLower(policy.Resource.Kind), hash)
	return namegen.ShortName(basename, validation.DNS1035LabelMaxLength)
}

func formatExternalName(googleTeamProjectId, resourceName string) string {
	if len(resourceName) == 0 {
		return fmt.Sprintf("projects/%s", googleTeamProjectId)
	}
	return fmt.Sprintf("projects/%s/%s", googleTeamProjectId, resourceName)
}

func CreatePolicyMember(source resource.Source, ast *resource.Ast, resourceOptions resource.Options, naisGcpPermission []nais.CloudIAMPermission) error {
	if naisGcpPermission == nil {
		return nil
	}

	for _, p := range naisGcpPermission {
		policyMember, err := policyMember(source, p, resourceOptions.GoogleProjectId, resourceOptions.GoogleTeamProjectId)
		if err != nil {
			return fmt.Errorf("unable to create iampolicymember: %w", err)
		}
		ast.AppendOperation(resource.OperationCreateIfNotExists, policyMember)
	}

	return nil
}

package google_iam

import (
	"fmt"
	"path"
	"strings"

	"github.com/mitchellh/hashstructure"
	google_iam_crd "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/namegen"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

type Source interface {
	resource.Source
	GetGCP() *nais_io_v1.GCP
}

type Config interface {
	GetGoogleProjectID() string
	GetGoogleTeamProjectID() string
}

func policyMember(source resource.Source, policy nais_io_v1.CloudIAMPermission, googleTeamProjectId string) (*google_iam_crd.IAMPolicyMember, error) {
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
			Member: fmt.Sprintf("serviceAccount:%s", google.GcpServiceAccountName(source.GetName(), googleTeamProjectId)),
			Role:   policy.Role,
			ResourceRef: google_iam_crd.ResourceRef{
				ApiVersion: policy.Resource.APIVersion,
				External:   &externalName,
				Kind:       policy.Resource.Kind,
			},
		},
	}

	return policyMember, nil
}

func createName(appName string, policy nais_io_v1.CloudIAMPermission) (string, error) {
	hash, err := hashstructure.Hash(policy, nil)
	if err != nil {
		return "", fmt.Errorf("while calculating hash from policy: %w", err)
	}
	basename := fmt.Sprintf("%s-%s-%x", appName, strings.ToLower(policy.Resource.Kind), hash)
	return namegen.ShortName(basename, validation.DNS1035LabelMaxLength)
}

func formatExternalName(googleTeamProjectId, resourceName string) string {
	projectPrefix := fmt.Sprintf("projects/%s", googleTeamProjectId)
	if len(resourceName) == 0 {
		return projectPrefix
	}

	if strings.HasPrefix(resourceName, projectPrefix) {
		return resourceName
	}

	return path.Join(projectPrefix, resourceName)
}

func CreatePolicyMember(source Source, ast *resource.Ast, cfg Config) error {
	gcp := source.GetGCP()
	if gcp == nil {
		return nil
	}

	for _, p := range gcp.Permissions {
		policyMember, err := policyMember(source, p, cfg.GetGoogleTeamProjectID())
		if err != nil {
			return fmt.Errorf("unable to create iampolicymember: %w", err)
		}
		ast.AppendOperation(resource.OperationCreateIfNotExists, policyMember)
	}

	return nil
}
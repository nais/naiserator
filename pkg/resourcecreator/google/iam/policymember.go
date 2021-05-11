package google_iam

import (
	"fmt"
	"strings"

	"github.com/mitchellh/hashstructure"
	google_iam_crd "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func GoogleIAMPolicyMember(app *nais.Application, policy nais.CloudIAMPermission, googleProjectId, googleTeamProjectId string) (*google_iam_crd.IAMPolicyMember, error) {
	name, err := createIAMPolicyMemberName(app, policy)
	externalName := formatIAMExternalName(googleTeamProjectId, policy.Resource.Name)
	if err != nil {
		return nil, err
	}
	policyMember := &google_iam_crd.IAMPolicyMember{
		ObjectMeta: (*app).CreateObjectMetaWithName(name),
		TypeMeta: metav1.TypeMeta{
			Kind:       "IAMPolicyMember",
			APIVersion: google.GoogleIAMAPIVersion,
		},
		Spec: google_iam_crd.IAMPolicyMemberSpec{
			Member: fmt.Sprintf("serviceAccount:%s", google.GcpServiceAccountName(app, googleProjectId)),
			Role:   policy.Role,
			ResourceRef: google_iam_crd.ResourceRef{
				ApiVersion: policy.Resource.APIVersion,
				External:   &externalName,
				Kind:       policy.Resource.Kind,
			},
		},
	}

	util.SetAnnotation(policyMember, google.GoogleProjectIdAnnotation, googleTeamProjectId)

	return policyMember, nil
}

func createIAMPolicyMemberName(app *nais.Application, policy nais.CloudIAMPermission) (string, error) {
	hash, err := hashstructure.Hash(policy, nil)
	if err != nil {
		return "", fmt.Errorf("while calculating hash from policy: %w", err)
	}
	basename := fmt.Sprintf("%s-%s-%x", app.Name, strings.ToLower(policy.Resource.Kind), hash)
	return namegen.ShortName(basename, validation.DNS1035LabelMaxLength)
}

func formatIAMExternalName(googleTeamProjectId, resourceName string) string {
	if len(resourceName) == 0 {
		return fmt.Sprintf("projects/%s", googleTeamProjectId)
	}
	return fmt.Sprintf("projects/%s/%s", googleTeamProjectId, resourceName)
}

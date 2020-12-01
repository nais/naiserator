package resourcecreator

import (
	"fmt"
	"strings"

	"github.com/mitchellh/hashstructure"
	google_iam_crd "github.com/nais/naiserator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const maxLengthResourceName = 63

func GoogleIAMPolicyMember(app *nais.Application, policy nais.CloudIAMPermission, googleProjectId, googleTeamProjectId string) (*google_iam_crd.IAMPolicyMember, error) {
	name, err := createIAMPolicyMemberName(app, policy)
	if err != nil {
		return nil, err
	}
	policyMember := &google_iam_crd.IAMPolicyMember{
		ObjectMeta: (*app).CreateObjectMetaWithName(name),
		TypeMeta: metav1.TypeMeta{
			Kind:       "IAMPolicyMember",
			APIVersion: GoogleIAMAPIVersion,
		},
		Spec: google_iam_crd.IAMPolicyMemberSpec{
			Member: fmt.Sprintf("serviceAccount:%s", GcpServiceAccountName(app, googleProjectId)),
			Role:   policy.Role,
			ResourceRef: google_iam_crd.ResourceRef{
				ApiVersion: policy.Resource.APIVersion,
				External:   fmt.Sprintf("projects/%s/%s", googleTeamProjectId, policy.Resource.Name),
				Kind:       policy.Resource.Kind,
			},
		},
	}

	setAnnotation(policyMember, GoogleProjectIdAnnotation, googleTeamProjectId)

	return policyMember, nil
}

func createIAMPolicyMemberName(app *nais.Application, policy nais.CloudIAMPermission) (string, error) {
	hash, err := hashstructure.Hash(policy, nil)
	if err != nil {
		return "", fmt.Errorf("while calculating hash from policy: %w", err)
	}
	basename := fmt.Sprintf("%s-%s-%x", app.Name, strings.ToLower(policy.Resource.Kind), hash)
	return util.StrShortName(basename, maxLengthResourceName)
}

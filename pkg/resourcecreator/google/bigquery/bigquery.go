package google_bigquery

import (
	"fmt"
	"strings"

	google_nais_io_v1 "github.com/nais/liberator/pkg/apis/google.nais.io/v1"
	google_iam_crd "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/pointer"
)

type Source interface {
	resource.Source
	GetGCP() *nais_io_v1.GCP
}

type Config interface {
	GetGoogleProjectID() string
	GetGoogleTeamProjectID() string
}

func CreateDataset(source Source, ast *resource.Ast, cfg Config, serviceAccountName string) error {
	gcp := source.GetGCP()
	if gcp == nil {
		return nil
	}

	for _, bigQuerySpec := range gcp.BigQueryDatasets {
		bigQueryInstance, err := createDataset(source, bigQuerySpec, cfg.GetGoogleTeamProjectID(), serviceAccountName)
		if err != nil {
			return err
		}
		ast.AppendOperation(resource.OperationCreateIfNotExists, bigQueryInstance)

		iamPolicyMember, err := iAMPolicyMember(source, bigQueryInstance, cfg.GetGoogleTeamProjectID(), serviceAccountName)
		if err != nil {
			return err
		}
		ast.AppendOperation(resource.OperationCreateIfNotExists, iamPolicyMember)
	}

	return nil
}

func iAMPolicyMember(source resource.Source, bigqueryDataset *google_nais_io_v1.BigQueryDataset, teamProjectID, serviceAccountName string) (*google_iam_crd.IAMPolicyMember, error) {
	shortName, err := namegen.ShortName(bigqueryDataset.Name+"-job-user", validation.DNS1035LabelMaxLength)
	if err != nil {
		return nil, err
	}
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = shortName
	util.SetAnnotation(&objectMeta, google.StateIntoSpec, google.StateIntoSpecValue)
	policy := &google_iam_crd.IAMPolicyMember{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IAMPolicyMember",
			APIVersion: google.IAMAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec: google_iam_crd.IAMPolicyMemberSpec{
			Member: fmt.Sprintf("serviceAccount:%s", google.GcpServiceAccountName(serviceAccountName, teamProjectID)),
			Role:   "roles/bigquery.jobUser",
			ResourceRef: google_iam_crd.ResourceRef{
				Kind: "Project",
				Name: pointer.StringPtr(""),
			},
		},
	}

	util.SetAnnotation(policy, google.ProjectIdAnnotation, teamProjectID)

	return policy, nil
}

func createDataset(source resource.Source, bigQuerySpec nais_io_v1.CloudBigQueryDataset, teamProjectID, serviceAccountName string) (*google_nais_io_v1.BigQueryDataset, error) {
	objectMeta := resource.CreateObjectMeta(source)
	datasetName := strings.ToLower(bigQuerySpec.Name)
	baseName := strings.ReplaceAll(fmt.Sprintf("%s-%s", source.GetName(), datasetName), "_", "-")

	shortName, err := namegen.ShortName(baseName, validation.DNS1035LabelMaxLength)
	if err != nil {
		return nil, err
	}
	objectMeta.Name = shortName

	cascadingDeleteAnnotationValue := "false"
	if bigQuerySpec.CascadingDelete {
		cascadingDeleteAnnotationValue = "true"
	}
	util.SetAnnotation(&objectMeta, google.CascadingDeleteAnnotation, cascadingDeleteAnnotationValue)

	return &google_nais_io_v1.BigQueryDataset{
		TypeMeta: metav1.TypeMeta{
			Kind:       "BigQueryDataset",
			APIVersion: google.BigQueryAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec: google_nais_io_v1.BigQueryDatasetSpec{
			Name:            datasetName,
			Location:        google.Region,
			Description:     bigQuerySpec.Description,
			Project:         teamProjectID,
			CascadingDelete: bigQuerySpec.CascadingDelete,
			Access: []google_nais_io_v1.DatasetAccess{
				{
					Role:        bigQuerySpec.Permission.GoogleType(),
					UserByEmail: google.GcpServiceAccountName(serviceAccountName, teamProjectID),
				},
			},
		},
	}, nil
}
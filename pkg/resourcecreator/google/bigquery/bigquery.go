package google_bigquery

import (
	"fmt"
	"strings"

	google_bigquery_crd "github.com/nais/liberator/pkg/apis/bigquery.cnrm.cloud.google.com/v1beta1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func CreateDataset(source resource.Source, ast *resource.Ast, resourceOptions resource.Options, bigQueryDatasets []nais_io_v1.CloudBigQueryDataset, serviceAccountName string) error {
	if bigQueryDatasets == nil {
		return nil
	}
	for _, bigQuerySpec := range bigQueryDatasets {
		bigQueryInstance, err := createDataset(source, bigQuerySpec, resourceOptions.GoogleProjectId, serviceAccountName)
		if err != nil {
			return err
		}
		ast.AppendOperation(resource.OperationCreateIfNotExists, bigQueryInstance)
	}
	return nil
}

func createDataset(source resource.Source, bigQuerySpec nais_io_v1.CloudBigQueryDataset, projectID, serviceAccountName string) (*google_bigquery_crd.BigQueryDataset, error) {
	objectMeta := resource.CreateObjectMeta(source)
	datasetName := strings.ToLower(bigQuerySpec.Name)
	baseName := strings.ReplaceAll(fmt.Sprintf("%s-%s", source.GetName(), datasetName), "_", "-")

	shortName, err := namegen.ShortName(baseName, validation.DNS1035LabelMaxLength)
	if err != nil {
		return nil, err
	}
	objectMeta.Name = shortName

	cascadingDeleteAnnotationValue := "false"
	if bigQuerySpec.CascadingDelete == true {
		cascadingDeleteAnnotationValue = "true"
	}
	util.SetAnnotation(&objectMeta, google.CascadingDeleteAnnotation, cascadingDeleteAnnotationValue)

	return &google_bigquery_crd.BigQueryDataset{
		TypeMeta: metav1.TypeMeta{
			Kind:       "BigQueryDataset",
			APIVersion: google.BigQueryAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec: google_bigquery_crd.BigqueryDatasetSpec{
			ResourceID:  datasetName,
			Location:    google.Region,
			Description: bigQuerySpec.Description,
			Access: []*google_bigquery_crd.BigQueryDatasetAccess{
				{
					Role:        bigQuerySpec.Permission.GoogleType(),
					UserByEmail: google.GcpServiceAccountName(serviceAccountName, projectID),
				},
			},
		},
	}, nil
}

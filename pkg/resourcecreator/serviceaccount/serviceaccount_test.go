package serviceaccount_test

import (
	"testing"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/serviceaccount"
	core "k8s.io/api/core/v1"

	"github.com/nais/naiserator/pkg/test/fixtures"

	"github.com/stretchr/testify/assert"
)

func TestGetServiceAccount(t *testing.T) {
	app := fixtures.MinimalApplication()
	ops := resource.Operations{}
	opts := resource.NewOptions()
	serviceaccount.Create(app.CreateObjectMeta(), opts, &ops, app.CreateAppNamespaceHash())
	svcAcc := ops[0].Resource.(*core.ServiceAccount)

	assert.Equal(t, app.Name, svcAcc.Name)
	assert.Equal(t, app.Namespace, svcAcc.Namespace)
}

func TestGetServiceAccountGoogleCluster(t *testing.T) {
	app := fixtures.MinimalApplication()
	ops := resource.Operations{}
	opts := resource.NewOptions()
	opts.GoogleProjectId = "nais-project-1234"
	serviceaccount.Create(app.CreateObjectMeta(), opts, &ops, app.CreateAppNamespaceHash())
	svcAcc := ops[0].Resource.(*core.ServiceAccount)

	assert.Equal(t, app.Name, svcAcc.Name)
	assert.Equal(t, app.Namespace, svcAcc.Namespace)
	assert.Equal(t, app.CreateAppNamespaceHash()+"@nais-project-1234.iam.gserviceaccount.com", svcAcc.Annotations["iam.gke.io/gcp-service-account"])
}

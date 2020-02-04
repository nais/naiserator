package resourcecreator_test

import (
	"fmt"
	"strings"
	"testing"

	google_iam_crd "github.com/nais/naiserator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	istio_networking_crd "github.com/nais/naiserator/pkg/apis/networking.istio.io/v1alpha3"
	google_sql_crd "github.com/nais/naiserator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	google_storage_crd "github.com/nais/naiserator/pkg/apis/storage.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	istio "istio.io/client-go/pkg/apis/security/v1beta1"
	"k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v1"
	core "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	networking "k8s.io/api/networking/v1"
	rbac "k8s.io/api/rbac/v1"
)

type realObjects struct {
	authorizationPolicy *istio.AuthorizationPolicy
	deployment          *v1.Deployment
	hpa                 *autoscaling.HorizontalPodAutoscaler
	ingress             *extensions.Ingress
	networkPolicy       *networking.NetworkPolicy
	role                *rbac.Role
	rolebinding         *rbac.RoleBinding
	secret              *core.Secret
	service             *core.Service
	serviceAccount      *core.ServiceAccount
	sqlDatabase             *google_sql_crd.SQLDatabase
	sqlInstance             *google_sql_crd.SQLInstance
	sqlUser                 *google_sql_crd.SQLUser
	virtualServices         []*istio_networking_crd.VirtualService
	googleIAMServiceAccount *google_iam_crd.IAMServiceAccount
	googleIAMPolicy         *google_iam_crd.IAMPolicy
	googleIAMPolicyMember   *google_iam_crd.IAMPolicyMember
	bucket                  *google_storage_crd.StorageBucket
	bucketAccessControl     *google_storage_crd.StorageBucketAccessControl
}

func getRealObjects(resources resourcecreator.ResourceOperations) (o realObjects) {
	for _, r := range resources {
		switch v := r.Resource.(type) {
		case *v1.Deployment:
			o.deployment = v
		case *core.Secret:
			o.secret = v
		case *core.Service:
			o.service = v
		case *core.ServiceAccount:
			o.serviceAccount = v
		case *autoscaling.HorizontalPodAutoscaler:
			o.hpa = v
		case *extensions.Ingress:
			o.ingress = v
		case *networking.NetworkPolicy:
			o.networkPolicy = v
		case *istio.AuthorizationPolicy:
			o.authorizationPolicy = v
		case *istio_networking_crd.VirtualService:
			o.virtualServices = append(o.virtualServices, v)
		case *rbac.Role:
			o.role = v
		case *rbac.RoleBinding:
			o.rolebinding = v
		case *google_iam_crd.IAMServiceAccount:
			o.googleIAMServiceAccount = v
		case *google_iam_crd.IAMPolicy:
			o.googleIAMPolicy = v
		case *google_iam_crd.IAMPolicyMember:
			o.googleIAMPolicyMember = v
		case *google_storage_crd.StorageBucket:
			o.bucket = v
		case *google_storage_crd.StorageBucketAccessControl:
			o.bucketAccessControl = v
		case *google_sql_crd.SQLInstance:
			o.sqlInstance = v
		case *google_sql_crd.SQLUser:
			o.sqlUser = v
		case *google_sql_crd.SQLDatabase:
			o.sqlDatabase = v
		}
	}
	return
}

// Test that a specified application spec results in the correct Kubernetes resources.
func TestCreate(t *testing.T) {
	t.Run("default application spec merges into empty struct", func(t *testing.T) {
		app := &nais.Application{}
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)
	})

	t.Run("application spec needs required parameters", func(t *testing.T) {
		app := fixtures.MinimalFailingApplication()
		opts := resourcecreator.NewResourceOptions()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.Error(t, err)
		assert.Nil(t, resources)
	})

	t.Run("team label and application name is propagated to created resources", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resourcecreator.NewResourceOptions()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.Equal(t, app.Name, objects.deployment.Name)
		assert.Equal(t, app.Namespace, objects.deployment.Namespace)
		assert.Equal(t, app.Name, objects.deployment.Labels["app"])
		assert.Equal(t, app.Labels["team"], objects.deployment.Labels["team"])
	})

	t.Run("all basic resource types are created from an application spec", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resourcecreator.NewResourceOptions()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources.Extract(resourcecreator.OperationCreateOrUpdate))
		assert.NotNil(t, objects.hpa)
		assert.NotNil(t, objects.service)
		assert.NotNil(t, objects.serviceAccount)
		assert.NotNil(t, objects.deployment)
		assert.Nil(t, objects.ingress)

		// Test that the Ingress is deleted
		objects = getRealObjects(resources.Extract(resourcecreator.OperationDeleteIfExists))
		assert.NotNil(t, objects.ingress)
	})

	t.Run("an ingress object is created if ingress paths are specified", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []string{"https://foo.bar/baz"}
		opts := resourcecreator.NewResourceOptions()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.NotNil(t, objects.ingress)
	})

	t.Run("erroneous ingress uris create errors", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []string{"gopher://lol"}
		opts := resourcecreator.NewResourceOptions()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.Error(t, err)
		assert.Nil(t, resources)
	})

	t.Run("istio resources are omitted when access policy creation is disabled", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resourcecreator.NewResourceOptions()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.Nil(t, objects.virtualServices)
		assert.Nil(t, objects.authorizationPolicy)
		assert.Nil(t, objects.networkPolicy)
	})

	t.Run("istio resources are created when access policy creation is enabled", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []string{"https://host.domain.tld"}
		opts := resourcecreator.NewResourceOptions()
		opts.AccessPolicy = true
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.Len(t, objects.virtualServices, 1)
		assert.NotNil(t, objects.authorizationPolicy)
		assert.NotNil(t, objects.networkPolicy)
	})

	t.Run("authorization policy resource are created when access policy creation is enabled", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resourcecreator.NewResourceOptions()
		opts.AccessPolicy = true
		app.Spec.Ingresses = []string{"https://foo.bar"}

		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.NotNil(t, objects.virtualServices)
		assert.NotNil(t, objects.authorizationPolicy)
		assert.NotNil(t, objects.networkPolicy)
	})

	t.Run("servicerolebinding and prometheus servicerolebinding resources are created when access policy creation is enabled", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resourcecreator.NewResourceOptions()
		opts.AccessPolicy = true
		app.Spec.AccessPolicy.Inbound.Rules = []nais.AccessPolicyRule{{"otherapp", "othernamespace"}}
		app.Spec.Prometheus.Enabled = true

		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.Equal(t, "cluster.local/ns/othernamespace/sa/otherapp", objects.authorizationPolicy.Spec.Rules[0].From[0].Source.Principals[0])
	})

	t.Run("leader election rbac is created when LE is requested", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.LeaderElection = true
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()
		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.NotNil(t, objects.role)
		assert.Equal(t, app.Name, objects.role.Name)
		assert.NotNil(t, objects.rolebinding)
		assert.Equal(t, app.Name, objects.rolebinding.Name)
	})

	t.Run("default network policy that allows egress to resources in kube-system and istio-system is created for app", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resourcecreator.NewResourceOptions()
		opts.AccessPolicy = true
		opts.AccessPolicyNotAllowedCIDRs = []string{"101.0.0.0/8"}
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.NotNil(t, objects.networkPolicy)

		assert.NotNil(t, objects.networkPolicy)
		assert.NotEmpty(t, objects.networkPolicy.Spec.Egress)
	})

	t.Run("omitting ingresses denies traffic from istio ingress gateway", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()
		opts.AccessPolicy = true

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		deletes := resources.Extract(resourcecreator.OperationDeleteIfExists)
		numDeletes := 0
		for _, resource := range deletes {
			switch res := resource.Resource.(type) {
			case *istio.AuthorizationPolicy:
				if res.GetName() == "myapplication" {
					numDeletes++
				}
			}
		}

		if numDeletes != 1 {
			t.Fail()
		}
	})

	t.Run("google service account, bucket, and bucket policy resources are coherent", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resourcecreator.NewResourceOptions()
		opts.GoogleProjectId = "nais-foo-1234"
		app.Spec.GCP = &nais.GCP{
			Buckets: []nais.CloudStorageBucket{
				{
					Name: "bucket-name",
				},
			},
		}

		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.NotNil(t, objects.googleIAMPolicy)
		assert.NotNil(t, objects.googleIAMServiceAccount)
		assert.NotNil(t, objects.bucketAccessControl)
		assert.NotNil(t, objects.bucket)

		entityTokens := strings.SplitN(objects.bucketAccessControl.Spec.Entity, "@", 2)

		// Requesting a bucket creates four separate Google resources.
		// There must be a connection between Bucket, Bucket IAM Policy, and Google Service Account.
		assert.True(t, strings.HasPrefix(objects.bucket.Name, app.Spec.GCP.Buckets[0].Name))
		assert.Equal(t, objects.bucket.Name, objects.bucketAccessControl.Spec.BucketRef.Name)
		assert.Equal(t, objects.bucket.Spec.Location, resourcecreator.GoogleRegion)
		assert.Equal(t, fmt.Sprintf("user-%s", objects.googleIAMServiceAccount.Name), entityTokens[0])
		assert.Equal(t, "nais-foo-1234.iam.gserviceaccount.com", entityTokens[1])

		assert.Equal(t, "abandon", objects.bucket.Annotations[resourcecreator.GoogleDeletionPolicyAnnotation])
	})

	t.Run("using gcp sqlinstance yields expected resources", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resourcecreator.NewResourceOptions()
		opts.GoogleProjectId = "nais-foo-1234"
		instanceName := app.Name
		dbName := "mydb"
		app.Spec.GCP = &nais.GCP{SqlInstances: []nais.CloudSqlInstance{
			{
				Type: nais.CloudSqlInstanceTypePostgres,
				Databases: []nais.CloudSqlDatabase{
					{
						Name: dbName,
					},
				},
			},
		}}

		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.NotNil(t, objects.googleIAMPolicy)
		assert.NotNil(t, objects.googleIAMServiceAccount)
		assert.NotNil(t, objects.sqlInstance)
		assert.NotNil(t, objects.sqlDatabase)
		assert.NotNil(t, objects.sqlUser)
		assert.NotNil(t, objects.secret)
		assert.NotNil(t, objects.googleIAMPolicyMember)

		assert.Equal(t, instanceName, objects.sqlInstance.Name)
		assert.Equal(t, "02:00", objects.sqlInstance.Spec.Settings.BackupConfiguration.StartTime)
		assert.Equal(t, app.Name, objects.sqlUser.Name)
		assert.Equal(t, dbName, objects.sqlDatabase.Name)
		assert.Equal(t, instanceName, objects.sqlDatabase.Spec.InstanceRef.Name)
		assert.Equal(t, instanceName, objects.sqlUser.Spec.InstanceRef.Name)
		assert.Equal(t, instanceName, objects.secret.StringData["GCP_SQLINSTANCE_MYAPPLICATION_USERNAME"])
		assert.Equal(t, instanceName, objects.googleIAMPolicyMember.Name)
		assert.Equal(t, fmt.Sprintf("sqlinstanceuser-%s", instanceName), objects.deployment.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.Name)
	})

}

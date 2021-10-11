package resourcecreator_test

import (
	"fmt"
	"strings"
	"testing"

	iam_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	sql_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	storage_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/storage.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v1"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	rbac "k8s.io/api/rbac/v1"
)

type realObjects struct {
	deployment              *appsv1.Deployment
	hpa                     *autoscaling.HorizontalPodAutoscaler
	ingress                 *networking.Ingress
	jwker                   *nais_io_v1.Jwker
	networkPolicy           *networking.NetworkPolicy
	role                    *rbac.Role
	rolebinding             *rbac.RoleBinding
	secret                  *core.Secret
	service                 *core.Service
	serviceAccount          *core.ServiceAccount
	sqlDatabase             *sql_cnrm_cloud_google_com_v1beta1.SQLDatabase
	sqlInstance             *sql_cnrm_cloud_google_com_v1beta1.SQLInstance
	sqlUser                 *sql_cnrm_cloud_google_com_v1beta1.SQLUser
	googleIAMServiceAccount *iam_cnrm_cloud_google_com_v1beta1.IAMServiceAccount
	googleIAMPolicy         *iam_cnrm_cloud_google_com_v1beta1.IAMPolicy
	googleIAMPolicyMember   *iam_cnrm_cloud_google_com_v1beta1.IAMPolicyMember
	bucket                  *storage_cnrm_cloud_google_com_v1beta1.StorageBucket
	bucketAccessControl     *storage_cnrm_cloud_google_com_v1beta1.StorageBucketAccessControl
}

func getRealObjects(resources resource.Operations) (o realObjects) {
	for _, r := range resources {
		switch v := r.Resource.(type) {
		case *appsv1.Deployment:
			o.deployment = v
		case *core.Secret:
			o.secret = v
		case *core.Service:
			o.service = v
		case *core.ServiceAccount:
			o.serviceAccount = v
		case *autoscaling.HorizontalPodAutoscaler:
			o.hpa = v
		case *networking.Ingress:
			o.ingress = v
		case *networking.NetworkPolicy:
			o.networkPolicy = v
		case *rbac.Role:
			o.role = v
		case *rbac.RoleBinding:
			o.rolebinding = v
		case *iam_cnrm_cloud_google_com_v1beta1.IAMServiceAccount:
			o.googleIAMServiceAccount = v
		case *iam_cnrm_cloud_google_com_v1beta1.IAMPolicy:
			o.googleIAMPolicy = v
		case *iam_cnrm_cloud_google_com_v1beta1.IAMPolicyMember:
			o.googleIAMPolicyMember = v
		case *storage_cnrm_cloud_google_com_v1beta1.StorageBucket:
			o.bucket = v
		case *storage_cnrm_cloud_google_com_v1beta1.StorageBucketAccessControl:
			o.bucketAccessControl = v
		case *sql_cnrm_cloud_google_com_v1beta1.SQLInstance:
			o.sqlInstance = v
		case *sql_cnrm_cloud_google_com_v1beta1.SQLUser:
			o.sqlUser = v
		case *sql_cnrm_cloud_google_com_v1beta1.SQLDatabase:
			o.sqlDatabase = v
		case *nais_io_v1.Jwker:
			o.jwker = v
		}
	}
	return
}

// Test that a specified application spec results in the correct Kubernetes resources.
func TestCreate(t *testing.T) {
	t.Run("default application spec merges into empty struct", func(t *testing.T) {
		app := &nais_io_v1alpha1.Application{}
		err := app.ApplyDefaults()
		assert.NoError(t, err)
	})

	t.Run("application spec needs required parameters", func(t *testing.T) {
		app := fixtures.MinimalFailingApplication()
		opts := resource.NewOptions()
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		resources, err := resourcecreator.CreateApplication(app, opts)
		assert.Error(t, err)
		assert.Nil(t, resources)
	})

	t.Run("team label and application name is propagated to created resources", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resource.NewOptions()
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		resources, err := resourcecreator.CreateApplication(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.Equal(t, app.Name, objects.deployment.Name)
		assert.Equal(t, app.Namespace, objects.deployment.Namespace)
		assert.Equal(t, app.Name, objects.deployment.Labels["app"])
		assert.Equal(t, app.Labels["team"], objects.deployment.Labels["team"])
	})

	t.Run("an ingress object is created if ingress paths are specified", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []nais_io_v1.Ingress{"https://foo.bar/baz"}
		opts := resource.NewOptions()
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		resources, err := resourcecreator.CreateApplication(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.NotNil(t, objects.ingress)
	})

	t.Run("erroneous ingress uris create errors", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []nais_io_v1.Ingress{"gopher://lol"}
		opts := resource.NewOptions()
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		resources, err := resourcecreator.CreateApplication(app, opts)
		assert.Error(t, err)
		assert.Nil(t, resources)
	})

	t.Run("jwker resource is not created when access policy is empty", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resource.NewOptions()
		err := app.ApplyDefaults()
		assert.NoError(t, err)
		resources, err := resourcecreator.CreateApplication(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.Empty(t, objects.jwker)
	})

	t.Run("network policies are created when access policy creation is enabled", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []nais_io_v1.Ingress{"https://host.domain.tld"}
		opts := resource.NewOptions()
		opts.GatewayMappings = []config.GatewayMapping{{DomainSuffix: ".domain.tld", IngressClass: "namespace/gateway"}}
		opts.NetworkPolicy = true
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		resources, err := resourcecreator.CreateApplication(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.NotNil(t, objects.networkPolicy)
	})

	t.Run("leader election rbac is created when LE is requested", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.LeaderElection = true
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		opts := resource.NewOptions()
		resources, err := resourcecreator.CreateApplication(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.NotNil(t, objects.role)
		assert.Equal(t, app.Name, objects.role.Name)
		assert.NotNil(t, objects.rolebinding)
		assert.Equal(t, app.Name, objects.rolebinding.Name)
	})

	t.Run("google service account, bucket, and bucket policy resources are coherent", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resource.NewOptions()
		opts.GoogleProjectId = "nais-foo-1234"
		opts.GoogleTeamProjectId = "team-project-id"
		opts.CNRMEnabled = true
		app.Spec.GCP = &nais_io_v1.GCP{
			Buckets: []nais_io_v1.CloudStorageBucket{
				{
					Name: "bucket-name",
				},
			},
		}

		err := app.ApplyDefaults()
		assert.NoError(t, err)

		resources, err := resourcecreator.CreateApplication(app, opts)
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
		assert.Equal(t, objects.bucket.Spec.Location, google.Region)
		assert.Equal(t, fmt.Sprintf("user-%s", objects.googleIAMServiceAccount.Name), entityTokens[0])
		assert.Equal(t, "nais-foo-1234.iam.gserviceaccount.com", entityTokens[1])

		assert.Equal(t, "abandon", objects.bucket.Annotations[google.DeletionPolicyAnnotation])
	})

	t.Run("using gcp sqlinstance yields expected resources", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resource.NewOptions()
		opts.CNRMEnabled = true
		opts.GoogleProjectId = "nais-foo-1234"
		opts.GoogleTeamProjectId = "team-project-id"
		instanceName := app.Name
		dbName := "mydb"
		app.Spec.GCP = &nais_io_v1.GCP{SqlInstances: []nais_io_v1.CloudSqlInstance{
			{
				Type: nais_io_v1.CloudSqlInstanceTypePostgres11,
				Databases: []nais_io_v1.CloudSqlDatabase{
					{
						Name: dbName,
					},
				},
			},
		}}

		err := app.ApplyDefaults()
		assert.NoError(t, err)

		resources, err := resourcecreator.CreateApplication(app, opts)
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
		assert.Equal(t, instanceName, objects.secret.StringData["NAIS_DATABASE_MYAPPLICATION_MYDB_USERNAME"])
		assert.Equal(t, instanceName, objects.googleIAMPolicyMember.Name)
		assert.Equal(t, fmt.Sprintf("google-sql-%s", instanceName), objects.deployment.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.Name)
		assert.True(t, objects.sqlInstance.Spec.Settings.IpConfiguration.RequireSsl)
	})

	t.Run("ensure that the ingresses and redirect URIs for idporten are valid", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.IDPorten = &nais_io_v1.IDPorten{Enabled: true}

		opts := resource.NewOptions()
		opts.DigdiratorEnabled = true

		_, err := resourcecreator.CreateApplication(app, opts)
		assert.Error(t, err, "return error if no ingresses are specified")

		app.Spec.Ingresses = []nais_io_v1.Ingress{
			"https://yolo-ingress.nais.io",
			"https://very-cool-ingress.nais.io",
		}
		_, err = resourcecreator.CreateApplication(app, opts)
		assert.Error(t, err, "return error if multiple ingresses are specified")

		app.Spec.Ingresses = []nais_io_v1.Ingress{
			"https://yolo-ingress.nais.io",
		}
		_, err = resourcecreator.CreateApplication(app, opts)
		assert.NoError(t, err, "should not return error if exactly one ingress specified")

		app.Spec.IDPorten.RedirectURI = "https://not-yolo.nais.io/oauth2/callback"
		_, err = resourcecreator.CreateApplication(app, opts)
		assert.Error(t, err, "return error if redirect URI is not subpath of ingress")

		app.Spec.Ingresses = []nais_io_v1.Ingress{
			"http://localhost/oauth2/callback",
		}
		app.Spec.IDPorten.RedirectURI = "http://localhost/oauth2/callback"
		_, err = resourcecreator.CreateApplication(app, opts)
		assert.Error(t, err, "return error if redirect URI and ingress does not start with https://")

		app.Spec.IDPorten.RedirectURI = "https://yolo-ingress.nais.io/oauth2/callback"
		app.Spec.Ingresses = []nais_io_v1.Ingress{
			"https://yolo-ingress.nais.io",
		}
		_, err = resourcecreator.CreateApplication(app, opts)
		assert.NoError(t, err, "should not return error if redirect URI is subpath of ingress")
	})
}

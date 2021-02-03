package resourcecreator_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/apis/networking.istio.io/v1alpha3"
	"github.com/nais/liberator/pkg/apis/security.istio.io/v1beta1"
	"github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/liberator/pkg/apis/storage.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v1"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	rbac "k8s.io/api/rbac/v1"
)

type realObjects struct {
	authorizationPolicy     *security_istio_io_v1beta1.AuthorizationPolicy
	deployment              *v1.Deployment
	hpa                     *autoscaling.HorizontalPodAutoscaler
	ingress                 *networkingv1beta1.Ingress
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
	virtualServices         []*networking_istio_io_v1alpha3.VirtualService
	googleIAMServiceAccount *iam_cnrm_cloud_google_com_v1beta1.IAMServiceAccount
	googleIAMPolicy         *iam_cnrm_cloud_google_com_v1beta1.IAMPolicy
	googleIAMPolicyMember   *iam_cnrm_cloud_google_com_v1beta1.IAMPolicyMember
	bucket                  *storage_cnrm_cloud_google_com_v1beta1.StorageBucket
	bucketAccessControl     *storage_cnrm_cloud_google_com_v1beta1.StorageBucketAccessControl
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
		case *networkingv1beta1.Ingress:
			o.ingress = v
		case *networking.NetworkPolicy:
			o.networkPolicy = v
		case *security_istio_io_v1beta1.AuthorizationPolicy:
			o.authorizationPolicy = v
		case *networking_istio_io_v1alpha3.VirtualService:
			o.virtualServices = append(o.virtualServices, v)
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
		err := nais_io_v1alpha1.ApplyDefaults(app)
		assert.NoError(t, err)
	})

	t.Run("application spec needs required parameters", func(t *testing.T) {
		app := fixtures.MinimalFailingApplication()
		opts := resourcecreator.NewResourceOptions()
		err := nais_io_v1alpha1.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.Error(t, err)
		assert.Nil(t, resources)
	})

	t.Run("team label and application name is propagated to created resources", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resourcecreator.NewResourceOptions()
		err := nais_io_v1alpha1.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.Equal(t, app.Name, objects.deployment.Name)
		assert.Equal(t, app.Namespace, objects.deployment.Namespace)
		assert.Equal(t, app.Name, objects.deployment.Labels["app"])
		assert.Equal(t, app.Labels["team"], objects.deployment.Labels["team"])
	})

	t.Run("an ingress object is created if ingress paths are specified", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{"https://foo.bar/baz"}
		opts := resourcecreator.NewResourceOptions()
		err := nais_io_v1alpha1.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.NotNil(t, objects.ingress)
	})

	t.Run("erroneous ingress uris create errors", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{"gopher://lol"}
		opts := resourcecreator.NewResourceOptions()
		err := nais_io_v1alpha1.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.Error(t, err)
		assert.Nil(t, resources)
	})

	t.Run("istio resources are omitted when access policy creation is disabled", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resourcecreator.NewResourceOptions()
		err := nais_io_v1alpha1.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.Nil(t, objects.virtualServices)
		assert.Nil(t, objects.authorizationPolicy)
		assert.Nil(t, objects.networkPolicy)
	})

	t.Run("jwker resource is not created when access policy is empty", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resourcecreator.NewResourceOptions()
		err := nais_io_v1alpha1.ApplyDefaults(app)
		assert.NoError(t, err)
		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.Empty(t, objects.jwker)
	})

	t.Run("istio resources are created when access policy creation is enabled", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{"https://host.domain.tld"}
		opts := resourcecreator.NewResourceOptions()
		opts.GatewayMappings = []config.GatewayMapping{{DomainSuffix: ".domain.tld", GatewayName: "namespace/gateway"}}
		opts.AccessPolicy = true
		err := nais_io_v1alpha1.ApplyDefaults(app)
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
		opts.GatewayMappings = []config.GatewayMapping{{DomainSuffix: ".bar", GatewayName: "namespace/gateway"}}
		app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{"https://foo.bar"}

		err := nais_io_v1alpha1.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.NotNil(t, objects.virtualServices)
		assert.NotNil(t, objects.authorizationPolicy)
		assert.NotNil(t, objects.networkPolicy)
	})

	t.Run("authorization policy resource are created when access policy creation is enabled", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resourcecreator.NewResourceOptions()
		opts.AccessPolicy = true
		app.Spec.AccessPolicy.Inbound.Rules = []nais_io_v1.AccessPolicyRule{{"otherapp", "othernamespace", ""}}
		app.Spec.Prometheus.Enabled = true

		err := nais_io_v1alpha1.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.Equal(t, "cluster.local/ns/othernamespace/sa/otherapp", objects.authorizationPolicy.Spec.Rules[1].From[0].Source.Principals[0])
	})

	t.Run("leader election rbac is created when LE is requested", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.LeaderElection = true
		err := nais_io_v1alpha1.ApplyDefaults(app)
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
		err := nais_io_v1alpha1.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.NotNil(t, objects.networkPolicy)

		assert.NotNil(t, objects.networkPolicy)
		assert.NotEmpty(t, objects.networkPolicy.Spec.Egress)
	})

	t.Run("google service account, bucket, and bucket policy resources are coherent", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resourcecreator.NewResourceOptions()
		opts.GoogleProjectId = "nais-foo-1234"
		app.Spec.GCP = &nais_io_v1alpha1.GCP{
			Buckets: []nais_io_v1alpha1.CloudStorageBucket{
				{
					Name: "bucket-name",
				},
			},
		}

		err := nais_io_v1alpha1.ApplyDefaults(app)
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
		app.Spec.GCP = &nais_io_v1alpha1.GCP{SqlInstances: []nais_io_v1alpha1.CloudSqlInstance{
			{
				Type: nais_io_v1alpha1.CloudSqlInstanceTypePostgres11,
				Databases: []nais_io_v1alpha1.CloudSqlDatabase{
					{
						Name: dbName,
					},
				},
			},
		}}

		err := nais_io_v1alpha1.ApplyDefaults(app)
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
		assert.Equal(t, instanceName, objects.secret.StringData["NAIS_DATABASE_MYAPPLICATION_MYDB_USERNAME"])
		assert.Equal(t, instanceName, objects.googleIAMPolicyMember.Name)
		assert.Equal(t, fmt.Sprintf("google-sql-%s", instanceName), objects.deployment.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.Name)
		assert.True(t, objects.sqlInstance.Spec.Settings.IpConfiguration.RequireSsl)
	})

	t.Run("ensure that the ingresses and redirect URIs for idporten are valid", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.IDPorten = &nais_io_v1alpha1.IDPorten{Enabled: true}

		opts := resourcecreator.NewResourceOptions()
		opts.DigdiratorEnabled = true

		_, err := resourcecreator.Create(app, opts)
		assert.Error(t, err, "return error if no ingresses are specified")

		app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
			"https://yolo-ingress.nais.io",
			"https://very-cool-ingress.nais.io",
		}
		_, err = resourcecreator.Create(app, opts)
		assert.Error(t, err, "return error if multiple ingresses are specified")

		app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
			"https://yolo-ingress.nais.io",
		}
		_, err = resourcecreator.Create(app, opts)
		assert.NoError(t, err, "should not return error if exactly one ingress specified")

		app.Spec.IDPorten.RedirectURI = "https://not-yolo.nais.io/oauth2/callback"
		_, err = resourcecreator.Create(app, opts)
		assert.Error(t, err, "return error if redirect URI is not subpath of ingress")

		app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
			"http://localhost/oauth2/callback",
		}
		app.Spec.IDPorten.RedirectURI = "http://localhost/oauth2/callback"
		_, err = resourcecreator.Create(app, opts)
		assert.Error(t, err, "return error if redirect URI and ingress does not start with https://")

		app.Spec.IDPorten.RedirectURI = "https://yolo-ingress.nais.io/oauth2/callback"
		app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
			"https://yolo-ingress.nais.io",
		}
		_, err = resourcecreator.Create(app, opts)
		assert.NoError(t, err, "should not return error if redirect URI is subpath of ingress")
	})
}

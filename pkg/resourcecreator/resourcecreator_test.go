package resourcecreator_test

import (
	"fmt"
	"strings"
	"testing"

	iam_cnrm_cloud_google_com_v1alpha1 "github.com/nais/naiserator/pkg/apis/iam.cnrm.cloud.google.com/v1alpha1"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	networking_istio_io_v1alpha3 "github.com/nais/naiserator/pkg/apis/networking.istio.io/v1alpha3"
	rbac_istio_io_v1alpha1 "github.com/nais/naiserator/pkg/apis/rbac.istio.io/v1alpha1"
	sqlv1alpha3 "github.com/nais/naiserator/pkg/apis/sql.cnrm.cloud.google.com/v1alpha3"
	storage_cnrm_cloud_google_com_v1alpha2 "github.com/nais/naiserator/pkg/apis/storage.cnrm.cloud.google.com/v1alpha2"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

type realObjects struct {
	deployment              *v1.Deployment
	hpa                     *autoscalingv1.HorizontalPodAutoscaler
	ingress                 *extensionsv1beta1.Ingress
	networkPolicy           *networkingv1.NetworkPolicy
	role                    *rbacv1.Role
	rolebinding             *rbacv1.RoleBinding
	secret                  *corev1.Secret
	service                 *corev1.Service
	serviceAccount          *corev1.ServiceAccount
	serviceRoleBindings     []*rbac_istio_io_v1alpha1.ServiceRoleBinding
	serviceRoles            []*rbac_istio_io_v1alpha1.ServiceRole
	sqlDatabase             *sqlv1alpha3.SQLDatabase
	sqlInstance             *sqlv1alpha3.SQLInstance
	sqlUser                 *sqlv1alpha3.SQLUser
	virtualServices         []*networking_istio_io_v1alpha3.VirtualService
	googleIAMServiceAccount *iam_cnrm_cloud_google_com_v1alpha1.IAMServiceAccount
	googleIAMPolicy         *iam_cnrm_cloud_google_com_v1alpha1.IAMPolicy
	bucket                  *storage_cnrm_cloud_google_com_v1alpha2.StorageBucket
	bucketAccessControl     *storage_cnrm_cloud_google_com_v1alpha2.StorageBucketAccessControl
}

func getRealObjects(resources resourcecreator.ResourceOperations) (o realObjects) {
	for _, r := range resources {
		switch v := r.Resource.(type) {
		case *v1.Deployment:
			o.deployment = v
		case *corev1.Secret:
			o.secret = v
		case *corev1.Service:
			o.service = v
		case *corev1.ServiceAccount:
			o.serviceAccount = v
		case *autoscalingv1.HorizontalPodAutoscaler:
			o.hpa = v
		case *extensionsv1beta1.Ingress:
			o.ingress = v
		case *networkingv1.NetworkPolicy:
			o.networkPolicy = v
		case *rbac_istio_io_v1alpha1.ServiceRole:
			o.serviceRoles = append(o.serviceRoles, v)
		case *rbac_istio_io_v1alpha1.ServiceRoleBinding:
			o.serviceRoleBindings = append(o.serviceRoleBindings, v)
		case *networking_istio_io_v1alpha3.VirtualService:
			o.virtualServices = append(o.virtualServices, v)
		case *rbacv1.Role:
			o.role = v
		case *rbacv1.RoleBinding:
			o.rolebinding = v
		case *iam_cnrm_cloud_google_com_v1alpha1.IAMServiceAccount:
			o.googleIAMServiceAccount = v
		case *iam_cnrm_cloud_google_com_v1alpha1.IAMPolicy:
			o.googleIAMPolicy = v
		case *storage_cnrm_cloud_google_com_v1alpha2.StorageBucket:
			o.bucket = v
		case *storage_cnrm_cloud_google_com_v1alpha2.StorageBucketAccessControl:
			o.bucketAccessControl = v
		case *sqlv1alpha3.SQLInstance:
			o.sqlInstance = v
		case *sqlv1alpha3.SQLUser:
			o.sqlUser = v
		case *sqlv1alpha3.SQLDatabase:
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
		assert.Nil(t, objects.serviceRoleBindings)
		assert.Nil(t, objects.serviceRoles)
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
		assert.NotNil(t, objects.serviceRoleBindings)
		assert.NotNil(t, objects.serviceRoles)
		assert.NotNil(t, objects.networkPolicy)
	})

	t.Run("servicerole and servicerolebinding resources are created when access policy creation is enabled", func(t *testing.T) {
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
		assert.NotNil(t, objects.serviceRoleBindings)
		assert.NotNil(t, objects.serviceRoles)
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
		assert.Len(t, objects.serviceRoles, 2)
		assert.Len(t, objects.serviceRoleBindings, 2)
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
			switch x := resource.Resource.(type) {
			case *rbac_istio_io_v1alpha1.ServiceRoleBinding:
				if x.GetName() == "myapplication" {
					numDeletes++
				}
			}
		}

		if numDeletes != 1 {
			t.Fail()
		}
	})

	t.Run("no service role and no service role binding created for prometheus, when disabled", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()

		app.Spec.Prometheus.Enabled = false
		opts.AccessPolicy = true

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		deletes := resources.Extract(resourcecreator.OperationDeleteIfExists)
		numDeletes := 0
		for _, resource := range deletes {
			switch x := resource.Resource.(type) {
			case *rbac_istio_io_v1alpha1.ServiceRoleBinding:
				if x.GetName() == "myapplication-prometheus" {
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
		assert.Equal(t, "bucket-name", objects.bucket.Name)
		assert.Equal(t, objects.bucket.Name, objects.bucketAccessControl.Spec.BucketRef.Name)
		assert.Equal(t, objects.googleIAMServiceAccount.Name, entityTokens[0])
		assert.Equal(t, "nais-foo-1234.iam.gserviceaccount.com", entityTokens[1])
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

		assert.Equal(t, instanceName, objects.sqlInstance.Name)
		assert.Equal(t, app.Name, objects.sqlUser.Name)
		assert.Equal(t, dbName, objects.sqlDatabase.Name)
		assert.Equal(t, instanceName, objects.sqlDatabase.Spec.InstanceRef.Name)
		assert.Equal(t, instanceName, objects.sqlUser.Spec.InstanceRef.Name)
		assert.Equal(t, instanceName, objects.secret.StringData["GCP_SQLINSTANCE_MYAPPLICATION_USERNAME"])
		assert.Equal(t, fmt.Sprintf("sqlinstanceuser-%s", instanceName), objects.deployment.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.Name)
	})

}

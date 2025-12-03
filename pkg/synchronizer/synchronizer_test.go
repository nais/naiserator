package synchronizer_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	go_runtime "runtime"
	"testing"
	"time"

	iam_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais_io "github.com/nais/liberator/pkg/apis/nais.io"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	sql_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/liberator/pkg/crd"
	"github.com/nais/liberator/pkg/events"
	liberator_scheme "github.com/nais/liberator/pkg/scheme"
	"github.com/nais/naiserator/pkg/controllers"
	"github.com/nais/naiserator/pkg/generators"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/ingress"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	resourcecreator_secret "github.com/nais/naiserator/pkg/resourcecreator/secret"
	naiserator_scheme "github.com/nais/naiserator/pkg/scheme"
	"github.com/nais/naiserator/pkg/synchronizer"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl_config "sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	correlationId = "my-correlation-id"
)

type testRig struct {
	kubernetes   *envtest.Environment
	client       client.Client
	manager      ctrl.Manager
	synchronizer reconcile.Reconciler
	scheme       *runtime.Scheme
	config       config.Config
}

// testResourceNotExists tests that a resource does not exist in the fake cluster
func (rig *testRig) testResourceNotExist(t *testing.T, ctx context.Context, resource client.Object, objectKey client.ObjectKey) {
	err := rig.client.Get(ctx, objectKey, resource)
	assert.True(t, errors.IsNotFound(err), "the resource found in the cluster should not be there")
}

// testResource tests that a resource has been created in the fake cluster
func (rig *testRig) testResource(t *testing.T, ctx context.Context, resource client.Object, objectKey client.ObjectKey, assertFuncs ...func(t *testing.T, resource client.Object)) {
	err := rig.client.Get(ctx, objectKey, resource)
	assert.NoError(t, err)
	assert.NotNil(t, resource)
	for _, assertFunc := range assertFuncs {
		assertFunc(t, resource)
	}
}

func testBinDirectory() string {
	_, filename, _, _ := go_runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "../../.testbin/"))
}

func newTestRig(config config.Config) (*testRig, error) {
	rig := &testRig{}

	crdPath := crd.YamlDirectory()
	rig.kubernetes = &envtest.Environment{
		CRDDirectoryPaths: []string{crdPath},
	}

	err := os.Setenv("KUBEBUILDER_ASSETS", testBinDirectory())
	if err != nil {
		return nil, fmt.Errorf("failed to set environment variable: %w", err)
	}

	rig.config = config

	cfg, err := rig.kubernetes.Start()
	if err != nil {
		return nil, fmt.Errorf("setup Kubernetes test environment: %w", err)
	}

	rig.scheme, err = liberator_scheme.All()
	if err != nil {
		return nil, fmt.Errorf("setup scheme: %w", err)
	}

	rig.client, err = client.New(cfg, client.Options{
		Scheme: rig.scheme,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize Kubernetes client: %w", err)
	}

	rig.manager, err = ctrl.NewManager(rig.kubernetes.Config, ctrl.Options{
		Controller: ctrl_config.Controller{
			SkipNameValidation: ptr.To(true),
		},
		Scheme: rig.scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("initialize manager: %w", err)
	}

	listers := naiserator_scheme.GenericListers()
	if len(rig.config.GoogleProjectId) > 0 {
		listers = append(listers, naiserator_scheme.GCPListers()...)

		if len(rig.config.AivenProject) > 0 {
			listers = append(listers, naiserator_scheme.AivenListers()...)
		}
	}

	applicationReconciler := controllers.NewAppReconciler(synchronizer.NewSynchronizer(
		rig.client,
		rig.client,
		rig.config,
		&generators.Application{
			Config: rig.config,
		},
		nil,
		listers,
		rig.scheme,
	))

	err = applicationReconciler.SetupWithManager(rig.manager)
	if err != nil {
		return nil, fmt.Errorf("setup synchronizer with manager: %w", err)
	}
	rig.synchronizer = applicationReconciler

	return rig, nil
}

// This test sets up a complete in-memory Kubernetes rig, and tests the reconciler (Synchronizer) against it.
// These tests ensure that resources are actually created or updated in the cluster,
// and that orphaned resources are cleaned up properly.
// The validity of resources generated are not tested here.
// This test includes some GCP features suchs as CNRM
func TestSynchronizer(t *testing.T) {
	cfg := config.Config{
		AivenGeneration: 0,
		Synchronizer: config.Synchronizer{
			SynchronizationTimeout: 2 * time.Second,
			RolloutCheckInterval:   5 * time.Second,
			RolloutTimeout:         20 * time.Second,
		},
		GoogleProjectId:                   "1337",
		GoogleCloudSQLProxyContainerImage: config.GoogleCloudSQLProxyContainerImage,
		Features: config.Features{
			CNRM: true,
		},
	}

	rig, err := newTestRig(cfg)
	if err != nil {
		t.Errorf("unable to run synchronizer integration tests: %s", err)
		t.FailNow()
	}

	defer rig.kubernetes.Stop()

	// Allow no more than 15 seconds for these tests to run
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	// Check that listing all resources work.
	// If this test fails, it might mean CRDs are not registered in the test rig.
	listers := naiserator_scheme.GenericListers()
	listers = append(listers, naiserator_scheme.GCPListers()...)
	listers = append(listers, naiserator_scheme.AivenListers()...)
	// DO NOT ADD! Adding the AcidZalandoListers here breaks the test due to some inconsistencies in how envtest responds to requests
	// listers = append(listers, naiserator_scheme.AcidZalandoListers()...)
	for _, list := range listers {
		err = rig.client.List(ctx, list)
		assert.NoError(t, err, "Unable to list resource, are the CRDs installed?")
	}

	// Ensure that the application's namespace exists
	err = rig.client.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fixtures.ApplicationNamespace,
		},
	})
	assert.NoError(t, err)

	// Ensure that the cnrm namespace exists
	err = rig.client.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: google.IAMServiceAccountNamespace,
		},
	})
	assert.NoError(t, err)

	t.Run("App Deployment", func(t *testing.T) {
		app := fixtures.MinimalApplication(
			fixtures.WithAnnotation(nais_io.DeploymentCorrelationIDAnnotation, "deploy-id"),
		)
		testAppDeployment(t, rig, ctx, app, cfg)

		rig.testResource(t, ctx, &nais_io_v1alpha1.Application{}, client.ObjectKey{Name: app.Name, Namespace: app.Namespace}, func(t *testing.T, resource client.Object) {
			app := resource.(*nais_io_v1alpha1.Application)
			assert.Equal(t, fixtures.DefaultApplicationImage, app.Status.EffectiveImage)
		})
	})

	t.Run("Delete IAM Resources", func(t *testing.T) {
		app := fixtures.MinimalApplication(
			fixtures.WithAnnotation(nais_io.DeploymentCorrelationIDAnnotation, "deploy-id"),
		)
		testDeleteCorrectIAMResources(t, rig, ctx, app)
	})

	t.Run("App With External Image Deployment", func(t *testing.T) {
		// Ensure that external image resource exists
		image := fixtures.MinimalImage(fixtures.WithName(fixtures.OtherApplicationName))
		err = rig.client.Create(ctx, image)
		assert.NoError(t, err)

		// Create Application fixture
		app := fixtures.MinimalApplication(
			fixtures.WithAnnotation(nais_io.DeploymentCorrelationIDAnnotation, "external-deploy-id"),
			appWithoutImage(),
			fixtures.WithName(fixtures.OtherApplicationName),
		)
		testAppDeployment(t, rig, ctx, app, cfg)

		rig.testResource(t, ctx, &nais_io_v1alpha1.Application{}, client.ObjectKey{Name: app.Name, Namespace: app.Namespace}, func(t *testing.T, resource client.Object) {
			app := resource.(*nais_io_v1alpha1.Application)
			assert.Equal(t, fixtures.OtherApplicationImage, app.Status.EffectiveImage)
		})
	})
}

func testAppDeployment(t *testing.T, rig *testRig, ctx context.Context, app *nais_io_v1alpha1.Application, cfg config.Config) {
	appCorrelationId := app.GetAnnotations()[nais_io.DeploymentCorrelationIDAnnotation]

	// Store the Application resource in the cluster before testing commences.
	// This simulates a deployment into the cluster which is then picked up by the
	// informer queue.
	err := rig.client.Create(ctx, app)
	if err != nil {
		t.Fatalf("Application resource cannot be persisted to fake Kubernetes: %s", err)
	}

	opts := &generators.Options{}
	opts.Config = cfg
	opts.Config.GatewayMappings = []config.GatewayMapping{
		{
			DomainSuffix: ".bar",
			IngressClass: "very-nginx",
		},
		{
			DomainSuffix: ".baz",
			IngressClass: "something-else",
		},
	}

	// Create an Ingress object that should be deleted once processing has run.
	ast := resource.NewAst()
	app.Spec.Ingresses = []nais_io_v1.Ingress{"https://foo.bar"}
	err = ingress.Create(app, ast, opts)
	assert.NoError(t, err)
	ing := ast.Operations[0].Resource.(*networkingv1.Ingress)
	app.Spec.Ingresses = []nais_io_v1.Ingress{}
	err = rig.client.Create(ctx, ing)
	if err != nil || len(ing.Spec.Rules) == 0 {
		t.Fatalf("BUG: error creating ingress for testing: %s", err)
	}

	// Create an Ingress object with application label but without ownerReference.
	// This resource should persist in the cluster even after synchronization.
	app.Spec.Ingresses = []nais_io_v1.Ingress{"https://foo.bar"}
	err = ingress.Create(app, ast, opts)
	assert.NoError(t, err)
	ing = ast.Operations[1].Resource.(*networkingv1.Ingress)
	disownedIngressName := "disowned-ingress-" + app.GetName()
	ing.SetName(disownedIngressName)
	ing.SetOwnerReferences(nil)
	app.Spec.Ingresses = []nais_io_v1.Ingress{}
	err = rig.client.Create(ctx, ing)
	if err != nil || len(ing.Spec.Rules) == 0 {
		t.Fatalf("BUG: error creating ingress 2 for testing: %s", err)
	}

	// Run synchronization processing.
	// This will attempt to store numerous resources in Kubernetes.
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: app.Namespace,
			Name:      app.Name,
		},
	}
	result, err := rig.synchronizer.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	objectKey := client.ObjectKey{Name: app.Name, Namespace: app.Namespace}
	persistedApp := &nais_io_v1alpha1.Application{}
	err = rig.client.Get(ctx, objectKey, persistedApp)
	assert.NoError(t, err)
	assert.Len(t, persistedApp.ObjectMeta.Finalizers, 1, "After the first reconcile only finalizer is set")

	// We need to run another reconcile after finalizer is set
	result, err = rig.synchronizer.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	// Test that the Application was updated successfully after processing,
	// that the team label is set, and that the hash is present.
	persistedApp = &nais_io_v1alpha1.Application{}
	err = rig.client.Get(ctx, objectKey, persistedApp)
	hash, _ := app.Hash(cfg.AivenGeneration)
	assert.NotNil(t, persistedApp)
	assert.Equal(t, app.Namespace, persistedApp.GetLabels()["team"], "Team label was added to the Application resource metadata")
	assert.NoError(t, err)
	assert.Equalf(t, hash, persistedApp.Status.SynchronizationHash, "Application resource hash in Kubernetes matches local version")

	// Test that the status field is set with RolloutComplete
	assert.Equalf(t, events.Synchronized, persistedApp.Status.SynchronizationState, "Synchronization state is set")
	assert.Equalf(t, appCorrelationId, persistedApp.Status.CorrelationID, "Correlation ID is set")

	// Test that a base resource set was created successfully
	rig.testResource(t, ctx, &appsv1.Deployment{}, objectKey)
	rig.testResource(t, ctx, &corev1.Service{}, objectKey)
	rig.testResource(t, ctx, &corev1.ServiceAccount{}, objectKey)

	// Test that the Ingress resource was removed
	rig.testResourceNotExist(t, ctx, &networkingv1.Ingress{}, objectKey)

	// Test that a Synchronized event was generated and has the correct deployment correlation id
	eventList := &corev1.EventList{}
	err = rig.client.List(ctx, eventList, client.MatchingLabels{"app": app.Name})
	assert.NoError(t, err)
	assert.Len(t, eventList.Items, 1)
	assert.EqualValues(t, 1, eventList.Items[0].Count)
	assert.Equal(t, appCorrelationId, eventList.Items[0].Annotations[nais_io.DeploymentCorrelationIDAnnotation])
	assert.Equal(t, events.Synchronized, eventList.Items[0].Reason)

	// Run synchronization processing again, and check that resources still exist.
	newAppCorrelationId := "new-" + appCorrelationId
	persistedApp.DeepCopyInto(app)
	app.Status.SynchronizationHash = ""
	app.Annotations[nais_io.DeploymentCorrelationIDAnnotation] = newAppCorrelationId
	err = rig.client.Update(ctx, app)
	assert.NoError(t, err)
	result, err = rig.synchronizer.Reconcile(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	rig.testResource(t, ctx, &appsv1.Deployment{}, objectKey, func(t *testing.T, resource client.Object) {
		dep := resource.(*appsv1.Deployment)
		assert.Equal(t, app.Name, dep.Name)
		assert.Equal(t, app.Status.EffectiveImage, dep.Spec.Template.Spec.Containers[0].Image)
	})
	rig.testResource(t, ctx, &corev1.Service{}, objectKey)
	rig.testResource(t, ctx, &corev1.ServiceAccount{}, objectKey)
	rig.testResource(t, ctx, &networkingv1.Ingress{}, client.ObjectKey{Name: disownedIngressName, Namespace: app.Namespace})

	// Test that the naiserator event was updated with increased count and new correlation id
	err = rig.client.List(ctx, eventList, client.MatchingLabels{"app": app.Name})
	assert.NoError(t, err)
	assert.Len(t, eventList.Items, 1)
	assert.EqualValues(t, 2, eventList.Items[0].Count)
	assert.Equal(t, newAppCorrelationId, eventList.Items[0].Annotations[nais_io.DeploymentCorrelationIDAnnotation])
	assert.Equal(t, events.Synchronized, eventList.Items[0].Reason)
}

func testDeleteCorrectIAMResources(t *testing.T, rig *testRig, ctx context.Context, app *nais_io_v1alpha1.Application) {
	app2 := fixtures.MinimalApplication()
	app2.SetAnnotations(map[string]string{
		nais_io.DeploymentCorrelationIDAnnotation: "deploy-id-2",
	})
	app2.ObjectMeta.Name = "iam-test"
	err := rig.client.Create(ctx, app2)
	if err != nil {
		t.Fatalf("Application resource cannot be persisted to fake Kubernetes: %s", err)
	}
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: app2.Namespace,
			Name:      app2.Name,
		},
	}
	// Reconcile for finalizer
	result, err := rig.synchronizer.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	// Reconcile to be synchronized
	result, err = rig.synchronizer.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	var iam_policies iam_cnrm_cloud_google_com_v1beta1.IAMPolicyList
	err = rig.client.List(ctx, &iam_policies)
	assert.NoError(t, err)
	assert.Len(t, iam_policies.Items, 2)

	var iam_service_accounts iam_cnrm_cloud_google_com_v1beta1.IAMServiceAccountList
	err = rig.client.List(ctx, &iam_service_accounts)
	assert.NoError(t, err)
	assert.Len(t, iam_service_accounts.Items, 2)
	assert.Equal(t, iam_service_accounts.Items[0].Labels["app"], app2.GetName())

	// Now, delete the application.
	// The application's children should disappear from the cluster.
	err = rig.client.Delete(ctx, app2)
	assert.NoError(t, err)

	req = ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: app2.Namespace,
			Name:      app2.Name,
		},
	}
	result, err = rig.synchronizer.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	err = rig.client.List(ctx, &iam_policies)
	assert.NoError(t, err)
	assert.Len(t, iam_policies.Items, 1)

	err = rig.client.List(ctx, &iam_service_accounts)
	assert.NoError(t, err)
	assert.Len(t, iam_service_accounts.Items, 1)
	assert.Equal(t, app.GetName(), iam_service_accounts.Items[0].Labels["app"])
}

func TestSynchronizerResourceOptions(t *testing.T) {
	cfg := config.Config{
		Synchronizer: config.Synchronizer{
			SynchronizationTimeout: 2 * time.Second,
			RolloutCheckInterval:   5 * time.Second,
			RolloutTimeout:         20 * time.Second,
		},
		Features: config.Features{
			CNRM: true,
		},
		GoogleProjectId:                   "something",
		GoogleCloudSQLProxyContainerImage: "cloudsqlproxy",
	}

	rig, err := newTestRig(cfg)
	if err != nil {
		t.Errorf("unable to run synchronizer integration tests: %s", err)
		t.FailNow()
	}

	defer rig.kubernetes.Stop()

	// Allow no more than 15 seconds for these tests to run
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	// Create Application fixture
	app := fixtures.MinimalApplication()
	app.SetAnnotations(map[string]string{
		nais_io.DeploymentCorrelationIDAnnotation: correlationId,
	})
	app.Spec.GCP = &nais_io_v1.GCP{
		SqlInstances: []nais_io_v1.CloudSqlInstance{
			{
				Type: nais_io_v1.CloudSqlInstanceTypePostgres17,
				Tier: "db-f1-micro",
				Databases: []nais_io_v1.CloudSqlDatabase{
					{
						Name: app.Name,
					},
				},
			},
		},
	}

	// Test that the team project id is fetched from namespace annotation, and used to create the sql proxy sidecar
	testProjectId := "test-project-id"
	testNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: app.GetNamespace(),
		},
	}
	testNamespace.SetAnnotations(map[string]string{
		google.ProjectIdAnnotation: testProjectId,
	})

	err = rig.client.Create(ctx, testNamespace)
	assert.NoError(t, err)

	// Ensure that namespace for Google IAM service accounts exists
	err = rig.client.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: google.IAMServiceAccountNamespace,
		},
	})
	assert.NoError(t, err)

	// Create a secret in the cluster that should get updated correlationId to trigger password sync
	googleSqlSecretName := fmt.Sprintf("google-sql-%s", app.GetName())
	objectMeta := metav1.ObjectMeta{
		Name:      googleSqlSecretName,
		Namespace: app.GetNamespace(),
	}
	existingGoogleSqlSecret := resourcecreator_secret.OpaqueSecret(objectMeta, googleSqlSecretName, nil)
	err = rig.client.Create(ctx, existingGoogleSqlSecret)
	assert.NoError(t, err)

	// Store the Application resource in the cluster before testing commences.
	// This simulates a deployment into the cluster which is then picked up by the
	// informer queue.
	err = rig.client.Create(ctx, app)
	if err != nil {
		t.Fatalf("Application resource cannot be persisted to fake Kubernetes: %s", err)
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: app.Namespace,
			Name:      app.Name,
		},
	}

	result, err := rig.synchronizer.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	// We need to run another reconcile after finalizer is set
	result, err = rig.synchronizer.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	deploy := &appsv1.Deployment{}
	sqlinstance := &sql_cnrm_cloud_google_com_v1beta1.SQLInstance{}
	sqluser := &sql_cnrm_cloud_google_com_v1beta1.SQLInstance{}
	sqldatabase := &sql_cnrm_cloud_google_com_v1beta1.SQLInstance{}
	iampolicymember := &iam_cnrm_cloud_google_com_v1beta1.IAMPolicyMember{}
	secret := &corev1.Secret{}

	err = rig.client.Get(ctx, req.NamespacedName, deploy)
	assert.NoError(t, err)
	expectedInstanceName := fmt.Sprintf("%s:%s:%s", testProjectId, google.Region, app.Name)
	cloudsqlProxyContainer := deploy.Spec.Template.Spec.InitContainers[0]
	actualInstanceNameFromCommand := cloudsqlProxyContainer.Command[6]
	assert.Equal(t, expectedInstanceName, actualInstanceNameFromCommand)

	err = rig.client.Get(ctx, req.NamespacedName, sqlinstance)
	assert.NoError(t, err)
	assert.Equal(t, testProjectId, sqlinstance.Annotations[google.ProjectIdAnnotation])

	err = rig.client.Get(ctx, req.NamespacedName, sqluser)
	assert.NoError(t, err)
	assert.Equal(t, testProjectId, sqluser.Annotations[google.ProjectIdAnnotation])

	err = rig.client.Get(ctx, req.NamespacedName, sqldatabase)
	assert.NoError(t, err)
	assert.Equal(t, testProjectId, sqldatabase.Annotations[google.ProjectIdAnnotation])

	err = rig.client.Get(ctx, req.NamespacedName, iampolicymember)
	assert.NoError(t, err)
	assert.Equal(t, testProjectId, iampolicymember.Annotations[google.ProjectIdAnnotation])

	err = rig.client.Get(ctx, types.NamespacedName{
		Namespace: req.Namespace,
		Name:      googleSqlSecretName,
	}, secret)
	assert.NoError(t, err)
	assert.Equal(t, correlationId, secret.Annotations[nais_io.DeploymentCorrelationIDAnnotation])

	// Simulate an Update event
	err = rig.client.Get(ctx, req.NamespacedName, app)
	require.NoError(t, err)

	newCorrelationId := "some-other-correlation-id"
	app.Annotations[nais_io.DeploymentCorrelationIDAnnotation] = newCorrelationId
	err = rig.client.Update(ctx, app)
	if err != nil {
		t.Fatalf("Persisting updated Application resource to fake Kubernetes: %s", err)
	}

	// We need to run another reconcile to simulate an Update event
	result, err = rig.synchronizer.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	updatedSqlUser := &sql_cnrm_cloud_google_com_v1beta1.SQLUser{}
	err = rig.client.Get(ctx, req.NamespacedName, updatedSqlUser)
	assert.NoError(t, err)
	assert.Equal(t, newCorrelationId, updatedSqlUser.Annotations[nais_io.DeploymentCorrelationIDAnnotation])

	updatedSecret := &corev1.Secret{}
	err = rig.client.Get(ctx, types.NamespacedName{
		Namespace: req.Namespace,
		Name:      googleSqlSecretName,
	}, updatedSecret)
	assert.NoError(t, err)
	assert.Equal(t, newCorrelationId, updatedSecret.Annotations[nais_io.DeploymentCorrelationIDAnnotation])
}

func appWithoutImage() fixtures.FixtureModifier {
	return func(obj client.Object) {
		app := obj.(*nais_io_v1alpha1.Application)
		app.Spec.Image = ""
	}
}

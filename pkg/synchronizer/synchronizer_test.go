package synchronizer_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nais/naiserator/pkg/controllers"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	ingress "github.com/nais/naiserator/pkg/resourcecreator/ingress"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"

	iam_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	sql_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/liberator/pkg/crd"
	liberator_scheme "github.com/nais/liberator/pkg/scheme"
	"github.com/nais/naiserator/pkg/naiserator/config"
	naiserator_scheme "github.com/nais/naiserator/pkg/scheme"
	"github.com/nais/naiserator/pkg/synchronizer"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type testRig struct {
	kubernetes   *envtest.Environment
	client       client.Client
	manager      ctrl.Manager
	synchronizer reconcile.Reconciler
	scheme       *runtime.Scheme
}

func newTestRig(options resource.Options) (*testRig, error) {
	rig := &testRig{}
	crdPath := crd.YamlDirectory()
	rig.kubernetes = &envtest.Environment{
		CRDDirectoryPaths: []string{crdPath},
	}

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
		Scheme:             rig.scheme,
		MetricsBindAddress: "0",
	})
	if err != nil {
		return nil, fmt.Errorf("initialize manager: %w", err)
	}

	syncerConfig := config.Config{
		Synchronizer: config.Synchronizer{
			SynchronizationTimeout: 2 * time.Second,
			RolloutCheckInterval:   5 * time.Second,
			RolloutTimeout:         20 * time.Second,
		},
	}

	applicationReconciler := controllers.NewAppReconciler(synchronizer.Synchronizer{
		Client:          rig.client,
		SimpleClient:    rig.client,
		Scheme:          rig.scheme,
		ResourceOptions: options,
		Config:          syncerConfig,
	})

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
func TestSynchronizer(t *testing.T) {
	resourceOptions := resource.NewOptions()
	rig, err := newTestRig(resourceOptions)
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
	for _, list := range listers {
		err = rig.client.List(ctx, list)
		assert.NoError(t, err)
	}

	// Create Application fixture
	app := fixtures.MinimalApplication()

	app.SetAnnotations(map[string]string{
		nais_io_v1.DeploymentCorrelationIDAnnotation: "deploy-id",
	})

	// Test that a resource has been created in the fake cluster
	testResource := func(resource runtime.Object, objectKey client.ObjectKey) {
		err := rig.client.Get(ctx, objectKey, resource)
		assert.NoError(t, err)
		assert.NotNil(t, resource)
	}

	// Test that a resource does not exist in the fake cluster
	testResourceNotExist := func(resource runtime.Object, objectKey client.ObjectKey) {
		err := rig.client.Get(ctx, objectKey, resource)
		assert.True(t, errors.IsNotFound(err), "the resource found in the cluster should not be there")
	}

	// Store the Application resource in the cluster before testing commences.
	// This simulates a deployment into the cluster which is then picked up by the
	// informer queue.
	err = rig.client.Create(ctx, app)
	if err != nil {
		t.Fatalf("Application resource cannot be persisted to fake Kubernetes: %s", err)
	}

	// Create an Ingress object that should be deleted once processing has run.
	ast := resource.NewAst()
	app.Spec.Ingresses = []nais_io_v1.Ingress{"https://foo.bar"}
	err = ingress.Create(app, ast, resourceOptions, app.Spec.Ingresses, app.Spec.Liveness.Path, app.Spec.Service.Protocol, app.Annotations)
	assert.NoError(t, err)
	ing := ast.Operations[0].Resource.(*networkingv1beta1.Ingress)
	app.Spec.Ingresses = []nais_io_v1.Ingress{}
	err = rig.client.Create(ctx, ing)
	if err != nil || len(ing.Spec.Rules) == 0 {
		t.Fatalf("BUG: error creating ingress for testing: %s", err)
	}

	// Create an Ingress object with application label but without ownerReference.
	// This resource should persist in the cluster even after synchronization.
	app.Spec.Ingresses = []nais_io_v1.Ingress{"https://foo.bar"}
	err = ingress.Create(app, ast, resourceOptions, app.Spec.Ingresses, app.Spec.Liveness.Path, app.Spec.Service.Protocol, app.Annotations)
	assert.NoError(t, err)
	ing = ast.Operations[1].Resource.(*networkingv1beta1.Ingress)
	ing.SetName("disowned-ingress")
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
	result, err := rig.synchronizer.Reconcile(req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	// Test that the Application was updated successfully after processing,
	// and that the hash is present.
	objectKey := client.ObjectKey{Name: app.Name, Namespace: app.Namespace}
	persistedApp := &nais_io_v1alpha1.Application{}
	err = rig.client.Get(ctx, objectKey, persistedApp)
	hash, _ := app.Hash()
	assert.NotNil(t, persistedApp)
	assert.NoError(t, err)
	assert.Equalf(t, hash, persistedApp.Status.SynchronizationHash, "Application resource hash in Kubernetes matches local version")

	// Test that the status field is set with RolloutComplete
	assert.Equalf(t, synchronizer.EventSynchronized, persistedApp.Status.SynchronizationState, "Synchronization state is set")
	assert.Equalf(t, "deploy-id", persistedApp.Status.CorrelationID, "Correlation ID is set")

	// Test that a base resource set was created successfully
	testResource(&appsv1.Deployment{}, objectKey)
	testResource(&corev1.Service{}, objectKey)
	testResource(&corev1.ServiceAccount{}, objectKey)

	// Test that the Ingress resource was removed
	testResourceNotExist(&networkingv1beta1.Ingress{}, objectKey)

	// Test that a Synchronized event was generated and has the correct deployment correlation id
	eventList := &corev1.EventList{}
	err = rig.client.List(ctx, eventList)
	assert.NoError(t, err)
	assert.Len(t, eventList.Items, 1)
	assert.EqualValues(t, 1, eventList.Items[0].Count)
	assert.Equal(t, "deploy-id", eventList.Items[0].Annotations[nais_io_v1.DeploymentCorrelationIDAnnotation])
	assert.Equal(t, synchronizer.EventSynchronized, eventList.Items[0].Reason)

	// Run synchronization processing again, and check that resources still exist.
	persistedApp.DeepCopyInto(app)
	app.Status.SynchronizationHash = ""
	app.Annotations[nais_io_v1.DeploymentCorrelationIDAnnotation] = "new-deploy-id"
	err = rig.client.Update(ctx, app)
	assert.NoError(t, err)
	result, err = rig.synchronizer.Reconcile(req)

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	testResource(&appsv1.Deployment{}, objectKey)
	testResource(&corev1.Service{}, objectKey)
	testResource(&corev1.ServiceAccount{}, objectKey)
	testResource(&networkingv1beta1.Ingress{}, client.ObjectKey{Name: "disowned-ingress", Namespace: app.Namespace})

	// Test that the naiserator event was updated with increased count and new correlation id
	err = rig.client.List(ctx, eventList)
	assert.NoError(t, err)
	assert.Len(t, eventList.Items, 1)
	assert.EqualValues(t, 2, eventList.Items[0].Count)
	assert.Equal(t, "new-deploy-id", eventList.Items[0].Annotations[nais_io_v1.DeploymentCorrelationIDAnnotation])
	assert.Equal(t, synchronizer.EventSynchronized, eventList.Items[0].Reason)
}

func TestSynchronizerResourceOptions(t *testing.T) {
	resourceOptions := resource.NewOptions()
	resourceOptions.GoogleProjectId = "something"
	resourceOptions.GoogleCloudSQLProxyContainerImage = "cloudsqlproxy"

	rig, err := newTestRig(resourceOptions)
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
	app.Spec.GCP = &nais_io_v1.GCP{
		SqlInstances: []nais_io_v1.CloudSqlInstance{
			{
				Type: nais_io_v1.CloudSqlInstanceTypePostgres11,
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

	result, err := rig.synchronizer.Reconcile(req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	deploy := &appsv1.Deployment{}
	sqlinstance := &sql_cnrm_cloud_google_com_v1beta1.SQLInstance{}
	sqluser := &sql_cnrm_cloud_google_com_v1beta1.SQLInstance{}
	sqldatabase := &sql_cnrm_cloud_google_com_v1beta1.SQLInstance{}
	iampolicymember := &iam_cnrm_cloud_google_com_v1beta1.IAMPolicyMember{}

	err = rig.client.Get(ctx, req.NamespacedName, deploy)
	assert.NoError(t, err)
	expectedInstanceName := fmt.Sprintf("-instances=%s:%s:%s=tcp:5432", testProjectId, google.Region, app.Name)
	assert.Equal(t, expectedInstanceName, deploy.Spec.Template.Spec.Containers[1].Command[2])

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
}

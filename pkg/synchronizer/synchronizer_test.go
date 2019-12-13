package synchronizer_test

import (
	"testing"

	nais_fake "github.com/nais/naiserator/pkg/client/clientset/versioned/fake"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/synchronizer"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// Test an entire synchronization run, i.e. create numerous resources
// from an Application resource.
func TestSynchronizer(t *testing.T) {
	// Create Application fixture
	app := fixtures.MinimalApplication()
	name := app.GetName()
	namespace := app.GetNamespace()

	// Test that a resource has been created in the fake cluster
	testResource := func(resource metav1.Object, err error) {
		assert.NoError(t, err)
		assert.NotNil(t, resource)
		assert.Equal(t, name, resource.GetName())
		assert.Equal(t, namespace, resource.GetNamespace())
	}

	// Test that a resource does not exist in the fake cluster
	testResourceNotExist := func(resource metav1.Object, err error) {
		assert.True(t, errors.IsNotFound(err), "the resource found in the cluster should not be there")
		assert.Nil(t, resource)
	}

	// Initialize synchronizer with fake Kubernetes clients
	clientSet := fake.NewSimpleClientset()
	appClient := nais_fake.NewSimpleClientset()
	resourceOptions := resourcecreator.NewResourceOptions()
	kafkaEnabled := false

	syncer := synchronizer.New(
		clientSet,
		appClient,
		resourceOptions,
		kafkaEnabled,
	)

	// Store the Application resource in the cluster before testing commences.
	// This simulates a deployment into the cluster which is then picked up by the
	// informer queue.
	app, err := appClient.NaiseratorV1alpha1().Applications(namespace).Create(app)
	if err != nil {
		t.Fatalf("Application resource cannot be persisted to fake Kubernetes: %s", err)
	}

	// Create an Ingress object that should be deleted once processing has run.
	app.Spec.Ingresses = []string{"https://foo.bar"}
	ingress, err := resourcecreator.Ingress(app)
	app.Spec.Ingresses = []string{}
	ingress, err = clientSet.ExtensionsV1beta1().Ingresses(namespace).Create(ingress)
	if err != nil || len(ingress.Spec.Rules) == 0 {
		t.Fatalf("BUG: error creating ingress for testing: %s", err)
	}

	// Run synchronization processing.
	// This will attempt to store numerous resources in Kubernetes.
	syncer.Process(app)

	// Test that the Application was updated successfully after processing,
	// and that the hash is present.
	persistedApp, err := appClient.NaiseratorV1alpha1().Applications(namespace).Get(name, metav1.GetOptions{})
	assert.NotNil(t, persistedApp)
	assert.NoError(t, err)
	assert.Equalf(t, app.LastSyncedHash(), persistedApp.LastSyncedHash(), "Application resource hash in Kubernetes matches local version")

	// Test that the status field is set with RolloutComplete
	assert.Equalf(t, synchronizer.EventSynchronized, persistedApp.Status.SynchronizationState, "Synchronization state is set")

	// Test that a base resource set was created successfully
	testResource(clientSet.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{}))
	testResource(clientSet.CoreV1().Services(namespace).Get(name, metav1.GetOptions{}))
	testResource(clientSet.CoreV1().ServiceAccounts(namespace).Get(name, metav1.GetOptions{}))

	// Test that the Ingress resource was removed
	testResourceNotExist(clientSet.ExtensionsV1beta1().Ingresses(namespace).Get(name, metav1.GetOptions{}))
}

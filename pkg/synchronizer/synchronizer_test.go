package synchronizer_test

import (
	"fmt"
	"testing"

	"k8s.io/api/core/v1"

	"github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
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
	app.SetAnnotations(map[string]string{
		v1alpha1.DeploymentCorrelationIDAnnotation: "deploy-id",
	})

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

	syncer := synchronizer.New(
		clientSet,
		appClient,
		resourceOptions,
		synchronizer.Config{
			KafkaEnabled: false,
		},
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
	assert.Equalf(t, app.Status.SynchronizationHash, persistedApp.Status.SynchronizationHash, "Application resource hash in Kubernetes matches local version")

	// Test that the status field is set with RolloutComplete
	assert.Equalf(t, synchronizer.EventSynchronized, persistedApp.Status.SynchronizationState, "Synchronization state is set")
	assert.Equalf(t, "deploy-id", persistedApp.Status.CorrelationID, "Correlation ID is set")

	// Test that a base resource set was created successfully
	testResource(clientSet.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{}))
	testResource(clientSet.CoreV1().Services(namespace).Get(name, metav1.GetOptions{}))
	testResource(clientSet.CoreV1().ServiceAccounts(namespace).Get(name, metav1.GetOptions{}))

	// Test that the Ingress resource was removed
	testResourceNotExist(clientSet.ExtensionsV1beta1().Ingresses(namespace).Get(name, metav1.GetOptions{}))
}

func TestSynchronizerResourceOptions(t *testing.T) {
	// Create Application fixture
	app := fixtures.MinimalApplication()
	app.Spec.GCP = &v1alpha1.GCP{SqlInstances: []v1alpha1.CloudSqlInstance{{}}}

	// Initialize synchronizer with fake Kubernetes clients
	clientSet := fake.NewSimpleClientset()
	appClient := nais_fake.NewSimpleClientset()
	resourceOptions := resourcecreator.NewResourceOptions()
	resourceOptions.GoogleProjectId = "something"

	syncer := synchronizer.New(
		clientSet,
		appClient,
		resourceOptions,
		synchronizer.Config{
			KafkaEnabled: false,
		},
	)

	// Test that the team project id is fetched from namespace annotation, and used to create the sql proxy sidecar
	testProjectId := "test-project-id"
	testNamespace := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: app.GetNamespace(),
		},
	}
	testNamespace.SetAnnotations(map[string]string{
		resourcecreator.GoogleProjectIdAnnotation: testProjectId,
	})

	_, err := clientSet.CoreV1().Namespaces().Create(&testNamespace)
	assert.NoError(t, err)

	syncer.Process(app)

	deploy, err := clientSet.AppsV1().Deployments(testNamespace.Name).Get(app.Name, metav1.GetOptions{})
	assert.NotNil(t, deploy)
	assert.NoError(t, err)

	expectedInstanceName := fmt.Sprintf("-instances=%s:%s:%s=tcp:5432", testProjectId, resourcecreator.GoogleRegion, app.Name)
	assert.Equal(t, expectedInstanceName, deploy.Spec.Template.Spec.Containers[1].Command[1])
}

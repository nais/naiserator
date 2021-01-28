package synchronizer_test

import (
	"fmt"
	"testing"

	"github.com/nais/liberator/pkg/crd"
	naiserator_scheme "github.com/nais/naiserator/pkg/naiserator/scheme"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/synchronizer"
	"github.com/stretchr/testify/assert"
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
}

func newTestRig() (*testRig, error) {
	rig := &testRig{}
	crdPath := crd.YamlDirectory()
	rig.kubernetes = &envtest.Environment{
		CRDDirectoryPaths: []string{crdPath},
	}

	cfg, err := rig.kubernetes.Start()
	if err != nil {
		return nil, fmt.Errorf("setup Kubernetes test environment: %w", err)
	}

	kscheme, err := naiserator_scheme.All()
	if err != nil {
		return nil, fmt.Errorf("setup scheme: %w", err)
	}

	rig.client, err = client.New(cfg, client.Options{
		Scheme: kscheme,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize Kubernetes client: %w", err)
	}

	rig.manager, err = ctrl.NewManager(rig.kubernetes.Config, ctrl.Options{
		Scheme:             kscheme,
		MetricsBindAddress: "0",
	})
	if err != nil {
		return nil, fmt.Errorf("initialize manager: %w", err)
	}

	syncerConfig := synchronizer.Config{}
	resourceOptions := resourcecreator.NewResourceOptions()

	syncer := &synchronizer.Synchronizer{
		Client:          rig.manager.GetClient(),
		ResourceOptions: resourceOptions,
		Config:          syncerConfig,
	}

	err = syncer.SetupWithManager(rig.manager)
	if err != nil {
		return nil, fmt.Errorf("setup synchronizer with manager: %w", err)
	}
	rig.synchronizer = syncer

	return rig, nil
}

func TestFoobar(t *testing.T) {
	rig, err := newTestRig()
	assert.NoError(t, err)
	assert.NotNil(t, rig)

	defer rig.kubernetes.Stop()

	go func() {
		err = rig.manager.Start(ctrl.SetupSignalHandler())
		if err != nil {
			panic(err)
		}
	}()

	// test here
}

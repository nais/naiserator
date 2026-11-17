package synchronizer_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	nais_io "github.com/nais/liberator/pkg/apis/nais.io"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/events"
	"github.com/nais/naiserator/pkg/generators"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/synchronizer"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

type failureStage int

const (
	noFailure failureStage = iota
	failPrepare
	failGenerate
	failSyncTransient
	failSyncPermanent
)

// failingClient wraps a client and fails the requested reconcile stage until fail is set to false.
type failingClient struct {
	client.WithWatch
	fail             bool
	deploymentWrites int
}

func newFailingClient(stage failureStage, base client.WithWatch) *failingClient {
	fc := &failingClient{fail: true}
	failDeploymentWrite := func(obj client.Object) error {
		if _, ok := obj.(*appsv1.Deployment); !ok {
			return nil
		}
		fc.deploymentWrites++
		if !fc.fail {
			return nil
		}
		switch stage {
		case failSyncTransient:
			return errors.NewConflict(appsv1.Resource("deployments"), obj.GetName(), fmt.Errorf("simulated conflict"))
		case failSyncPermanent:
			return errors.NewBadRequest("simulated rejection")
		}
		return nil
	}
	fc.WithWatch = interceptor.NewClient(base, interceptor.Funcs{
		Get: func(
			ctx context.Context, c client.WithWatch,
			key client.ObjectKey, obj client.Object, opts ...client.GetOption,
		) error {
			if _, ok := obj.(*appsv1.Deployment); ok && fc.fail && stage == failPrepare {
				return fmt.Errorf("deployment lookup unavailable")
			}
			return c.Get(ctx, key, obj, opts...)
		},
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			if err := failDeploymentWrite(obj); err != nil {
				return err
			}
			return c.Create(ctx, obj, opts...)
		},
		Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			if err := failDeploymentWrite(obj); err != nil {
				return err
			}
			return c.Update(ctx, obj, opts...)
		},
	})
	return fc
}

func deploymentWithImage(app *nais_io_v1alpha1.Application, image string) *appsv1.Deployment {
	labels := map[string]string{"app": app.Name}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            app.Name,
			Namespace:       app.Namespace,
			OwnerReferences: []metav1.OwnerReference{app.GetOwnerReference()},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: app.Name, Image: image}},
				},
			},
		},
	}
}

// assertNextRolloutUsesImage checks that, once the failure is gone, the image change is still detected and
// the generated Deployment uses the wanted image.
func assertNextRolloutUsesImage(
	t *testing.T, sync *synchronizer.Synchronizer, generator synchronizer.Generator, kube client.Client,
	key client.ObjectKey, image string,
) {
	t.Helper()
	persisted := &nais_io_v1alpha1.Application{}
	require.NoError(t, kube.Get(t.Context(), key, persisted))
	persisted.Spec.Observability = nil

	rollout, err := sync.Prepare(t.Context(), persisted)
	require.NoError(t, err)
	require.NotNil(t, rollout, "image change must not be skipped after a failed reconcile")
	operations, err := generator.Generate(rollout.Source, rollout.Options)
	require.NoError(t, err)
	assert.Equal(t, image, persisted.Status.EffectiveImage)

	deployments := 0
	for _, operation := range operations {
		if generated, ok := operation.Resource.(*appsv1.Deployment); ok {
			deployments++
			assert.Equal(t, image, generated.Spec.Template.Spec.Containers[0].Image)
		}
	}
	require.Equal(t, 1, deployments)
}

func TestReconcileEffectiveImage(t *testing.T) {
	cfg := config.Config{
		Synchronizer:  config.Synchronizer{SynchronizationTimeout: 5 * time.Second},
		Observability: config.Observability{Otel: config.Otel{Enabled: true}},
	}
	rig, err := newTestRig(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rig.kubernetes.Stop() })

	base, err := client.NewWithWatch(rig.kubernetes.Config, client.Options{Scheme: rig.scheme})
	require.NoError(t, err)
	require.NoError(t, rig.client.Create(t.Context(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: fixtures.ApplicationNamespace},
	}))

	for i, tt := range []struct {
		name          string
		stage         failureStage
		previousImage string
		externalImage bool
	}{
		{name: "successful rollout", stage: noFailure, previousImage: fixtures.DefaultApplicationImage},
		{name: "permanent sync failure", stage: failSyncPermanent, previousImage: fixtures.DefaultApplicationImage},
		{name: "preparation failure", stage: failPrepare, previousImage: fixtures.DefaultApplicationImage},
		{
			name:          "generation failure with external image",
			stage:         failGenerate,
			previousImage: fixtures.DefaultApplicationImage,
			externalImage: true,
		},
		{name: "first deployment generation failure", stage: failGenerate},
		{
			name:          "transient sync failure with external image",
			stage:         failSyncTransient,
			previousImage: fixtures.DefaultApplicationImage,
			externalImage: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			// All cases share one API server, so each gets its own application.
			name := fmt.Sprintf("app-%d", i)
			app := fixtures.MinimalApplication(
				fixtures.WithName(name),
				fixtures.WithAnnotation(nais_io.DeploymentCorrelationIDAnnotation, correlationId),
			)
			app.Finalizers = []string{synchronizer.NaiseratorFinalizer}
			// Without the team label, Reconcile adds it after hashing, which forces one extra resync.
			app.Labels = map[string]string{"team": app.Namespace}
			app.Spec.Image = fixtures.OtherApplicationImage
			if tt.stage == failGenerate {
				// Auto-instrumentation is not supported by the test config, which makes generation fail.
				app.Spec.Observability = &nais_io_v1.Observability{
					AutoInstrumentation: &nais_io_v1.AutoInstrumentation{Enabled: true, Runtime: "java"},
				}
			}
			if tt.externalImage {
				app.Spec.Image = ""
				require.NoError(t, rig.client.Create(ctx, fixtures.MinimalImage(fixtures.WithName(name))))
			}
			require.NoError(t, rig.client.Create(ctx, app))
			app.Status.EffectiveImage = tt.previousImage
			require.NoError(t, rig.client.Status().Update(ctx, app))
			if tt.previousImage != "" {
				require.NoError(t, rig.client.Create(ctx, deploymentWithImage(app, tt.previousImage)))
			}
			key := client.ObjectKeyFromObject(app)

			kube := newFailingClient(tt.stage, base)
			generator := &generators.Application{Config: cfg}
			sync := synchronizer.NewSynchronizer(kube, kube, cfg, generator, nil, rig.scheme)

			wantImage := tt.previousImage
			if tt.stage == noFailure || tt.stage == failSyncPermanent {
				wantImage = fixtures.OtherApplicationImage
			}

			// Reconcile twice: a failure before a final outcome must be retried, a final outcome must not be.
			for range 2 {
				_, err := sync.Reconcile(ctx, ctrl.Request{NamespacedName: key}, &nais_io_v1alpha1.Application{})
				persisted := &nais_io_v1alpha1.Application{}
				require.NoError(t, rig.client.Get(ctx, key, persisted))
				switch tt.stage {
				case noFailure:
					require.NoError(t, err)
					assert.Equal(t, events.Synchronized, persisted.Status.SynchronizationState)
				case failSyncPermanent:
					require.NoError(t, err)
					assert.Equal(t, events.FailedSynchronization, persisted.Status.SynchronizationState)
					assert.NotEmpty(t, persisted.Status.SynchronizationHash)
				case failPrepare:
					require.NoError(t, err)
					assert.Equal(t, events.FailedPrepare, persisted.Status.SynchronizationState)
				case failGenerate:
					require.ErrorContains(t, err, "auto-instrumentation is not supported for this cluster")
					assert.Equal(t, events.FailedGenerate, persisted.Status.SynchronizationState)
				case failSyncTransient:
					require.ErrorContains(t, err, "simulated conflict")
					assert.Equal(t, events.Retrying, persisted.Status.SynchronizationState)
				}
				assert.Equal(t, wantImage, persisted.Status.EffectiveImage)
			}

			switch tt.stage {
			case noFailure, failSyncPermanent:
				assert.Equal(t, 1, kube.deploymentWrites, "a final outcome must not be retried")
			case failSyncTransient:
				assert.Equal(t, 2, kube.deploymentWrites, "a transient failure must be retried")
			default:
				assert.Zero(t, kube.deploymentWrites)
			}

			kube.fail = false
			deployment := &appsv1.Deployment{}
			err := rig.client.Get(ctx, key, deployment)
			switch {
			case tt.stage == noFailure:
				require.NoError(t, err)
				assert.Equal(t, fixtures.OtherApplicationImage, deployment.Spec.Template.Spec.Containers[0].Image)
				return
			case tt.previousImage == "":
				assert.True(t, errors.IsNotFound(err))
			default:
				require.NoError(t, err)
				assert.Equal(t, tt.previousImage, deployment.Spec.Template.Spec.Containers[0].Image)
			}
			if tt.stage == failSyncPermanent {
				return
			}

			assertNextRolloutUsesImage(t, sync, generator, rig.client, key, fixtures.OtherApplicationImage)
		})
	}
}

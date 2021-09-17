package synchronizer

import (
	"context"
	"fmt"
	"time"

	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	"github.com/nais/naiserator/pkg/metrics"
	"github.com/nais/naiserator/pkg/resourcecreator/deployment"
	"github.com/nais/naiserator/pkg/resourcecreator/horizontalpodautoscaler"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/poddisruptionbudget"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/service"
	"github.com/nais/naiserator/pkg/resourcecreator/serviceaccount"
	"github.com/nais/naiserator/pkg/skatteetaten_generator/authorization_policy"
	"github.com/nais/naiserator/pkg/skatteetaten_generator/image_policy"
	"github.com/nais/naiserator/pkg/skatteetaten_generator/network_policy"
	"github.com/nais/naiserator/pkg/skatteetaten_generator/postgres"
	"github.com/nais/naiserator/pkg/skatteetaten_generator/service_entry"
	"github.com/nais/naiserator/pkg/skatteetaten_generator/virtual_service"
	log "github.com/sirupsen/logrus"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileSkatteetatenApplication process Application work queue
func (n *Synchronizer) ReconcileSkatteetatenApplication(req ctrl.Request) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), n.Config.Synchronizer.SynchronizationTimeout)
	defer cancel()

	app := &skatteetaten_no_v1alpha1.Application{}
	err := n.Get(ctx, req.NamespacedName, app)
	if err != nil {
		if errors.IsNotFound(err) {
			logger := log.WithFields(log.Fields{
				"namespace":   req.Namespace,
				"application": req.Name,
			})
			logger.Infof("Application has been deleted from Kubernetes")

			err = nil
		}
		return ctrl.Result{}, err
	}

	changed := true

	logger := *log.WithFields(app.LogFields())

	// Update Application resource with deployment event
	defer func() {
		if !changed {
			return
		}
		err := n.UpdateSkatteetatenApplication(ctx, app, func(existing *skatteetaten_no_v1alpha1.Application) error {
			existing.Status = app.Status
			return n.Update(ctx, app)
		})
		if err != nil {
			n.reportError(ctx, EventFailedStatusUpdate, err, app)
		} else {
			logger.Debugf("Application status: %+v'", app.Status)
		}
	}()

	rollout, err := n.PrepareSkatteetatenApplikasjon(app)
	if err != nil {
		app.Status.SynchronizationState = EventFailedPrepare
		n.reportError(ctx, app.Status.SynchronizationState, err, app)
		return ctrl.Result{RequeueAfter: prepareRetryInterval}, nil
	}

	if rollout == nil {
		changed = false
		logger.Debugf("Synchronization hash not changed; skipping synchronization")

		// Application is not rolled out completely; start monitoring
		if app.Status.SynchronizationState == EventSynchronized {
			n.MonitorRolloutSkatteetaten(app, logger)
		}

		return ctrl.Result{}, nil
	}

	logger = *log.WithFields(app.LogFields())
	logger.Debugf("Starting Skatteetaten synchronization")
	metrics.ApplicationsProcessed.Inc()

	app.Status.CorrelationID = rollout.CorrelationID

	retry, err := n.Sync(ctx, *rollout)
	if err != nil {
		if retry {
			app.Status.SynchronizationState = EventRetrying
			metrics.ApplicationsRetries.Inc()
			n.reportError(ctx, app.Status.SynchronizationState, err, app)
		} else {
			app.Status.SynchronizationState = EventFailedSynchronization
			app.Status.SynchronizationHash = rollout.SynchronizationHash // permanent failure
			metrics.ApplicationsFailed.Inc()
			n.reportError(ctx, app.Status.SynchronizationState, err, app)
			err = nil
		}
		return ctrl.Result{}, err
	}

	// Synchronization OK
	logger.Debugf("Successful synchronization")
	app.Status.SynchronizationState = EventSynchronized
	app.Status.SynchronizationHash = rollout.SynchronizationHash
	app.Status.SynchronizationTime = time.Now().UnixNano()
	metrics.Deployments.Inc()

	_, err = n.reportEvent(ctx, resource.CreateEvent(app, app.Status.SynchronizationState, "Successfully synchronized all application resources", "Normal"))
	if err != nil {
		log.Errorf("While creating an event for this rollout, an error occurred: %s", err)
	}

	// Monitor the rollout status so that we can report a successfully completed rollout to NAIS deploy.
	n.MonitorRolloutSkatteetaten(app, logger)

	return ctrl.Result{}, nil
}

// Prepare converts a NAIS application spec into a Rollout object.
// This is a read-only operation
// The Rollout object contains callback functions that commits changes in the cluster.
func (n *Synchronizer) PrepareSkatteetatenApplikasjon(app *skatteetaten_no_v1alpha1.Application) (*Rollout, error) {
	ctx := context.Background()
	var err error

	rollout := &Rollout{
		Source:          app,
		ResourceOptions: n.ResourceOptions,
	}

	  if err = app.ApplyDefaults(); err != nil {
		return nil, fmt.Errorf("BUG: merge default values into application: %s", err)
	  }

	rollout.SynchronizationHash, err = app.Hash()
	if err != nil {
		return nil, fmt.Errorf("BUG: create application hash: %s", err)
	}

	// Skip processing if application didn't change since last synchronization.
	if app.Status.SynchronizationHash == rollout.SynchronizationHash {
		return nil, nil
	}

	err = app.EnsureCorrelationID()
	if err != nil {
		return nil, err
	}

	rollout.CorrelationID = app.CorrelationID()

	// Make a query to Kubernetes for this application's previous deployment.
	// The number of replicas is significant, so we need to carry it over to match
	// this next rollout.
	previousDeployment := &apps.Deployment{}
	err = n.Get(ctx, client.ObjectKey{Name: app.GetName(), Namespace: app.GetNamespace()}, previousDeployment)
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("query existing deployment: %s", err)
	}

	// Retrieve current namespace to check for labels and annotations
	namespace := &corev1.Namespace{}
	err = n.Get(ctx, client.ObjectKey{Name: app.GetNamespace()}, namespace)
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("query existing namespace: %s", err)
	}

	rollout.SetCurrentDeployment(previousDeployment, *app.Spec.Replicas.Min)
	rollout.ResourceOperations, err = CreateSkatteetatenApplication(app, rollout.ResourceOptions)

	if err != nil {
		return nil, fmt.Errorf("creating cluster resource operations: %s", err)
	}

	return rollout, nil
}

// UpdateSkatteetatenApplication atomically update an Application resource.
// Locks the resource to avoid race conditions.
func (n *Synchronizer) UpdateSkatteetatenApplication(ctx context.Context, source resource.Source, updateFunc func(existing *skatteetaten_no_v1alpha1.Application) error) error {
	appsync.Lock()
	defer appsync.Unlock()

	existing := &skatteetaten_no_v1alpha1.Application{}
	err := n.Get(ctx, client.ObjectKey{Namespace: source.GetNamespace(), Name: source.GetName()}, existing)
	if err != nil {
		return fmt.Errorf("get newest version of Application: %s", err)
	}

	return updateFunc(existing)
}


func CreateSkatteetatenApplication(app *skatteetaten_no_v1alpha1.Application, resourceOptions resource.Options) (resource.Operations, error) {

	ast := resource.NewAst()

	naisApp := app.ToNaisApplication()

	service.Create(app, ast, resourceOptions, *naisApp.Spec.Service)
	serviceaccount.Create(app, ast, resourceOptions)
	horizontalpodautoscaler.CreateV1(app, ast, *app.Spec.Replicas)

	if !app.Spec.UnsecureDebugDisableAllAccessPolicies {
		network_policy.Create(app, ast, app.Spec)
		authorization_policy.Create(app, ast, app.Spec)
	}

	service_entry.Create(app, ast, app.Spec.Egress)
	virtual_service.Create(app, ast, app.Spec.Ingress)
	poddisruptionbudget.Create(app, ast, *app.Spec.Replicas)

	// TODO: Denne er i et annet ns så kan ikke ha owner reference, hvordan får vi slettet ting da?
	err := image_policy.Create(app, ast, app.Spec.ImagePolicy)
	if err != nil {
		return nil, err
	}

	postgres.Create(app, ast, app.Spec.Azure.PostgreDatabases, app.Spec.Azure.ResourceGroup)
	err = pod.CreateAppContainer(naisApp, ast, resourceOptions)
	if err != nil {
		return nil, err
	}

	err = deployment.Create(naisApp, ast, resourceOptions)
	if err != nil {
		return nil, err
	}

	return ast.Operations, nil
}
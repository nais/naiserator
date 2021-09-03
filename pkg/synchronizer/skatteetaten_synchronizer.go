package synchronizer

import (
	"context"
	"fmt"
	"time"

	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	"github.com/nais/naiserator/pkg/metrics"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	generator "github.com/nais/naiserator/pkg/skatteetaten"
	log "github.com/sirupsen/logrus"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (

	RESOURCE_GROUP ="rg-nap"
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
			//TODO: do not add monitor for now
			//n.MonitorRollout(app, logger)
		}

		return ctrl.Result{}, nil
	}

	logger = *log.WithFields(app.LogFields())
	logger.Debugf("Starting synchronization")
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
	//n.MonitorRollout(app, logger)

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

	//if err = app.ApplyDefaults(); err != nil {
	//	return nil, fmt.Errorf("BUG: merge default values into application: %s", err)
	//}

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

	rollout.SetCurrentDeployment(previousDeployment, app.Spec.Replicas.Min)
	rollout.ResourceOperations, err = CreateSkatteetatenApplication(app, rollout.ResourceOptions)

	if err != nil {
		return nil, fmt.Errorf("creating cluster resource operations: %s", err)
	}

	return rollout, nil
}



// UpdateApplication atomically update an Application resource.
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

	// Service
	svc := generator.GenerateService(*app)
	ast.AppendOperation(resource.OperationCreateOrUpdate, svc)

	// ServiceAccount
	sa := generator.GenerateServiceAccount(*app)
	ast.AppendOperation(resource.OperationCreateIfNotExists, sa)

	// HorizontalPodAutoscaler
	if app.Spec.Replicas.Min != app.Spec.Replicas.Max {
		hpa := generator.GenerateHpa(*app)
		ast.AppendOperation(resource.OperationCreateOrUpdate, hpa)
	}

	if ! app.Spec.UnsecureDebugDisableAllAccessPolicies {
		// NetworkPolicy
		np := generator.GenerateNetworkPolicy(*app, app.Spec)
		ast.AppendOperation(resource.OperationCreateOrUpdate, np)

		// AuthorizationPolicy
		ap := generator.GenerateAuthorizationPolicy(*app, app.Spec)
		ast.AppendOperation(resource.OperationCreateOrUpdate, ap)
	}

	// ServiceEntry
	if app.Spec.Egress != nil && app.Spec.Egress.External != nil {
		for _, egress := range app.Spec.Egress.External {
			se := generator.GenerateServiceEntry(*app, egress)
			ast.AppendOperation(resource.OperationCreateOrUpdate, se)
		}
	}

	// VirtualService
	if app.Spec.Ingress != nil && app.Spec.Ingress.Public != nil {
		for _, ingress := range app.Spec.Ingress.Public {
			if !ingress.Enabled {
				continue
			}
			vs := generator.GenerateVirtualService(*app, ingress)
			ast.AppendOperation(resource.OperationCreateOrUpdate, vs)
		}
	}

	// PodDisruptionBudget
	poddisruptionbudget := generator.GeneratePodDisruptionBudget(*app)
	ast.AppendOperation(resource.OperationCreateOrUpdate, poddisruptionbudget)

	// ImagePolicy
	imagePolicy, err := generator.GenerateImagePolicy(*app)
	if err != nil {
		return nil, err
	}
	ast.AppendOperation(resource.OperationCreateOrUpdate, imagePolicy)

	// Azure
	var dbVars []corev1.EnvVar
	if app.Spec.Azure != nil && app.Spec.Azure.PostgreDatabases != nil && len(app.Spec.Azure.PostgreDatabases) == 1 {
		dbVars = generator.GenerateDbEnv("SPRING_DATASOURCE", app.Spec.Azure.PostgreDatabases[0].Users[0].SecretName(*app))
	}

	//TODO handle updating
	for _, db := range app.Spec.Azure.PostgreDatabases {
		//TODO: handle fetching resource group from azure, or how do we do this?
		postgreDatabase := generator.GeneratePostgresDatabase(*app, RESOURCE_GROUP, *db)
		ast.AppendOperation(resource.OperationCreateIfNotExists, postgreDatabase)

		for _, user := range db.Users {
			postgreUser := generator.GeneratePostgresUser(*app, RESOURCE_GROUP, *db, *user)
			ast.AppendOperation(resource.OperationCreateIfNotExists, postgreUser)
		}
	}

	// Deployment
	deployment := generator.GenerateDeployment(*app, dbVars)
	ast.AppendOperation(resource.OperationCreateOrUpdate, deployment)


	return ast.Operations, nil
}
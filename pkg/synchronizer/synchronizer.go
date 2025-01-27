package synchronizer

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	iam_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/liberator/pkg/events"

	"github.com/nais/naiserator/pkg/event/generator"
	"github.com/nais/naiserator/pkg/kafka"
	"github.com/nais/naiserator/pkg/metrics"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/readonly"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/updater"
)

const (
	prepareRetryInterval = time.Minute * 30
	NaiseratorFinalizer  = "naiserator.nais.io/finalizer"
)

// Generator transform CRD objects such as Application, Naisjob into other kinds of Kubernetes resources.
// First, `Prepare()` is called. This function has access to (read-only) cluster operations and returns
// a configuration object. Then, `Generate()` is called with the configuration object, and returns a full
// set of Kubernetes resources.
type Generator interface {
	Prepare(ctx context.Context, source resource.Source, kube client.Client) (interface{}, error)
	Generate(source resource.Source, options interface{}) (resource.Operations, error)
}

// Synchronizer creates child resources from Application resources in the cluster.
// If the child resources does not match the Application spec, the resources are updated.
type Synchronizer struct {
	client.Client
	config         config.Config
	generator      Generator
	kafka          kafka.Interface
	listers        []client.ObjectList
	rolloutMonitor map[client.ObjectKey]RolloutMonitor
	scheme         *runtime.Scheme
	simpleClient   client.Client
	listableCache  map[schema.GroupVersionKind]bool
}

func NewSynchronizer(
	cli client.Client,
	simpleClient client.Client,
	config config.Config,
	generator Generator,
	kafka kafka.Interface,
	listers []client.ObjectList,
	scheme *runtime.Scheme,
) *Synchronizer {

	rolloutMonitor := make(map[client.ObjectKey]RolloutMonitor)
	return &Synchronizer{
		Client:         cli,
		config:         config,
		generator:      generator,
		kafka:          kafka,
		listers:        listers,
		rolloutMonitor: rolloutMonitor,
		scheme:         scheme,
		simpleClient:   simpleClient,
		listableCache:  make(map[schema.GroupVersionKind]bool),
	}
}

// Commit wraps a cluster operation function with extra fields
type commit struct {
	groupVersionKind schema.GroupVersionKind
	fn               func() error
}

// Creates a Kubernetes event, or updates an existing one with an incremented counter
func (n *Synchronizer) reportEvent(ctx context.Context, reportedEvent *corev1.Event) (*corev1.Event, error) {
	selector, err := fields.ParseSelector(fmt.Sprintf("involvedObject.name=%s,involvedObject.uid=%s", reportedEvent.InvolvedObject.Name, reportedEvent.InvolvedObject.UID))
	if err != nil {
		return nil, fmt.Errorf("internal error: unable to parse query: %s", err)
	}
	events := &corev1.EventList{}
	err = n.simpleClient.List(ctx, events, &client.ListOptions{
		FieldSelector: selector,
	})
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("get events for app '%s': %s", reportedEvent.InvolvedObject.Name, err)
	}

	for _, event := range events.Items {
		if event.Message == reportedEvent.Message {
			event.Count++
			event.LastTimestamp = reportedEvent.LastTimestamp
			event.SetAnnotations(reportedEvent.GetAnnotations())
			return &event, n.Update(ctx, &event)
		}
	}

	err = n.Create(ctx, reportedEvent)
	if err != nil {
		return nil, err
	}
	return reportedEvent, nil
}

// Reports an error through the error log, a Kubernetes event, and possibly logs a failure in event creation.
func (n *Synchronizer) reportError(ctx context.Context, eventSource string, err error, source resource.Source) {
	logger := log.WithFields(source.LogFields())
	logger.Error(err)
	_, err = n.reportEvent(ctx, resource.CreateEvent(source, eventSource, err.Error(), "Warning"))
	if err != nil {
		logger.Errorf("While creating an event for this error, another error occurred: %s", err)
	}
}

// Reconcile processes the work queue
func (n *Synchronizer) Reconcile(ctx context.Context, req ctrl.Request, app resource.Source) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, n.config.Synchronizer.SynchronizationTimeout)
	defer cancel()

	err := n.Get(ctx, req.NamespacedName, app)
	if err != nil {
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// We have some old, fun data in the conditions array from before. Zero them out for now to avoid confusion.
	// fixme: can be removed when we don't have garbage data anymore. Late 2024?
	app.GetStatus().Conditions = nil

	// Clear out any old problems from previous synchronizations.
	app.GetStatus().ClearProblems()

	kind := app.GetObjectKind().GroupVersionKind().Kind
	changed := true

	logger := *log.WithFields(app.LogFields())

	// Update Application resource with deployment event
	defer func() {
		if !changed {
			return
		}
		metrics.Synchronizations.With(prometheus.Labels{
			"kind":   kind,
			"status": app.GetStatus().SynchronizationState,
			"team":   app.GetNamespace(),
		}).Inc()
		err := n.UpdateResource(ctx, app, func(existing resource.Source) error {
			existing.SetStatus(app.GetStatus())

			// The team name was previously required as `.metadata.labels.team`.
			// Teams have had their own namespace many years now, but this team label
			// have persisted still. When this requirement was removed as of this patch,
			// users may deploy resources where the team label is not set.
			//
			// This function adds the team label to the resource again, in case there
			// are systems that use the label, especially the ones we don't control.
			labels := existing.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels["team"] = existing.GetNamespace()
			existing.SetLabels(labels)

			return n.Update(ctx, existing) // was app
		})
		if err != nil {
			n.reportError(ctx, events.FailedStatusUpdate, err, app)
		} else {
			logger.Debugf("Application status: %+v'", app.GetStatus())
		}
	}()

	if appIsDeleted(app) {
		err = n.cleanUpAfterAppDeletion(ctx, app)
		if err != nil {
			return ctrl.Result{}, err
		}

		logger := log.WithFields(log.Fields{
			"namespace": req.Namespace,
			"name":      req.Name,
			"gvk":       app.GetObjectKind().GroupVersionKind().String(),
		})
		logger.Infof("Application has been deleted from Kubernetes")

		changed = false // don't run update after deletion
		return ctrl.Result{}, nil
	} else {
		if !controllerutil.ContainsFinalizer(app, NaiseratorFinalizer) {
			controllerutil.AddFinalizer(app, NaiseratorFinalizer)
			err = n.Update(ctx, app)
			if err != nil {
				return ctrl.Result{}, err
			}
			changed = false // don't run update after finalizer is set
			return ctrl.Result{}, nil
		}
	}

	// Prepare configuration
	rollout, err := n.Prepare(ctx, app)
	if err != nil {
		app.GetStatus().SetSynchronizationStateWithCondition(events.FailedPrepare, err.Error())
		app.GetStatus().SetError(err.Error())
		n.reportError(ctx, app.GetStatus().SynchronizationState, err, app)
		return ctrl.Result{RequeueAfter: prepareRetryInterval}, nil
	}

	if rollout == nil {
		changed = false
		logger.Debugf("Synchronization hash not changed; skipping synchronization")

		// Application is not rolled out completely; start monitoring
		if app.GetStatus().SynchronizationState == events.Synchronized {
			src, ok := app.(generator.MonitorSource)
			if ok {
				n.MonitorRollout(src, logger)
			}
		}

		return ctrl.Result{}, nil
	}

	// Generate the actual Kubernetes resources that are going out into the cluster
	rollout.ResourceOperations, err = n.generator.Generate(rollout.Source, rollout.Options)
	if err != nil {
		app.GetStatus().SetSynchronizationStateWithCondition(events.FailedGenerate, err.Error())
		app.GetStatus().SetError(err.Error())
		n.reportError(ctx, app.GetStatus().SynchronizationState, err, app)
		return ctrl.Result{}, err
	}

	logger = *log.WithFields(app.LogFields())
	logger.Debugf("Starting synchronization")

	app.GetStatus().CorrelationID = rollout.CorrelationID

	retry, err := n.Sync(ctx, *rollout)
	if err != nil {
		if retry {
			app.GetStatus().SetSynchronizationStateWithCondition(events.Retrying, err.Error())
			// No error in problems array, because this is a transient error which the user has no control over.
			n.reportError(ctx, app.GetStatus().SynchronizationState, err, app)
		} else {
			app.GetStatus().SetSynchronizationStateWithCondition(events.FailedSynchronization, err.Error())
			app.GetStatus().SetError(err.Error())
			app.GetStatus().SynchronizationHash = rollout.SynchronizationHash // permanent failure
			n.reportError(ctx, app.GetStatus().SynchronizationState, err, app)
			err = nil
		}
		return ctrl.Result{}, err
	}

	// Synchronization OK
	syncMsg := "Deployment has been processed; waiting for completion..."
	logger.Debugf("Successful synchronization")
	app.GetStatus().SetSynchronizationStateWithCondition(events.Synchronized, syncMsg)
	app.GetStatus().SynchronizationHash = rollout.SynchronizationHash
	app.GetStatus().SynchronizationTime = time.Now().UnixNano()

	_, err = n.reportEvent(ctx, resource.CreateEvent(app, app.GetStatus().SynchronizationState, syncMsg, "Normal"))
	if err != nil {
		log.Errorf("While creating an event for this rollout, an error occurred: %s", err)
	}

	// Monitor the rollout status so that we can report a successfully completed rollout to NAIS deploy.
	src, ok := app.(generator.MonitorSource)
	if ok {
		n.MonitorRollout(src, logger)
	}

	return ctrl.Result{}, nil
}

func (n *Synchronizer) cleanUpAfterAppDeletion(ctx context.Context, app resource.Source) error {
	if controllerutil.ContainsFinalizer(app, NaiseratorFinalizer) {
		err := n.deleteCNRMResources(ctx, app)
		if err != nil {
			return err
		}

		controllerutil.RemoveFinalizer(app, NaiseratorFinalizer)
		err = n.Update(ctx, app)
		if err != nil {
			return err
		}
	}

	return nil
}

func appIsDeleted(app resource.Source) bool {
	return !app.GetObjectMeta().GetDeletionTimestamp().IsZero()
}

// deleteCNRMResources removes the lingering IAMServiceAccounts and IAMPolicies in the serviceaccounts namespace
func (n *Synchronizer) deleteCNRMResources(ctx context.Context, app resource.Source) error {
	if !n.config.Features.CNRM {
		return nil
	}

	labelSelector := labels.NewSelector()
	appLabelreq, err := labels.NewRequirement("app", selection.Equals, []string{app.GetName()})
	if err != nil {
		return err
	}
	labelSelector = labelSelector.Add(*appLabelreq)
	teamLabelreq, err := labels.NewRequirement("team", selection.Equals, []string{app.GetNamespace()})
	if err != nil {
		return err
	}
	labelSelector = labelSelector.Add(*teamLabelreq)
	listOpts := &client.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     google.IAMServiceAccountNamespace,
	}

	IAMServiceAccountList := &iam_cnrm_cloud_google_com_v1beta1.IAMServiceAccountList{}
	err = n.List(ctx, IAMServiceAccountList, listOpts)
	if err != nil {
		return err
	}

	for _, item := range IAMServiceAccountList.Items {
		err = n.Delete(ctx, &item)
		if err != nil {
			return err
		}
	}

	IAMPolicies := &iam_cnrm_cloud_google_com_v1beta1.IAMPolicyList{}
	err = n.List(ctx, IAMPolicies, listOpts)
	if err != nil {
		return err
	}

	for _, item := range IAMPolicies.Items {
		err = n.Delete(ctx, &item)
		if err != nil {
			return err
		}
	}

	return nil
}

// Unreferenced return all resources in cluster which was created by synchronizer previously, but is not included in the current rollout.
func (n *Synchronizer) Unreferenced(ctx context.Context, rollout Rollout) ([]runtime.Object, error) {
	// Return true if a cluster resource also is applied with the rollout.
	intersects := func(existing runtime.Object) bool {
		existingMeta, err := meta.Accessor(existing)
		if err != nil {
			log.Errorf("BUG: unable to determine TypeMeta for existing resource: %s", err)
			return true
		}
		for _, rop := range rollout.ResourceOperations {
			// Normally we would use GroupVersionKind to compare resource types, but due to
			// https://github.com/kubernetes/client-go/issues/308 the GVK is not set on the existing resource.
			// Reflection seems to work fine here.
			resourceMeta, err := meta.Accessor(rop.Resource)
			if err != nil {
				log.Errorf("BUG: unable to determine TypeMeta for new resource: %s", err)
				return true
			}
			if reflect.TypeOf(rop.Resource) == reflect.TypeOf(existing) {
				if resourceMeta.GetName() == existingMeta.GetName() {
					return true
				}
			}
		}
		return false
	}

	resources, err := updater.FindAll(ctx, n.Client, n.listers, rollout.Source)
	if err != nil {
		return nil, fmt.Errorf("discovering unreferenced resources: %s", err)
	}

	unreferenced := make([]runtime.Object, 0, len(resources))
	for _, existing := range resources {
		if !intersects(existing) {
			unreferenced = append(unreferenced, existing)
		}
	}

	return unreferenced, nil
}

func (n *Synchronizer) rolloutWithRetryAndMetrics(commits []commit) (bool, error) {
	for _, commit := range commits {
		if err := observeDuration(commit.fn); err != nil {
			retry := false
			// In case of race condition errors
			if errors.IsConflict(err) {
				retry = true
			}
			reason := errors.ReasonForError(err)
			if reason == metav1.StatusReasonUnknown {
				reason = "validation error"
			}
			return retry, fmt.Errorf("persisting %s to Kubernetes: %s: %s", commit.groupVersionKind.Kind, reason, err)
		}
		metrics.ResourcesGenerated.WithLabelValues(commit.groupVersionKind.Kind).Inc()
	}
	return false, nil
}

func (n *Synchronizer) Sync(ctx context.Context, rollout Rollout) (bool, error) {
	commits := n.ClusterOperations(ctx, rollout)
	return n.rolloutWithRetryAndMetrics(commits)
}

// Prepare converts a NAIS application spec into a Rollout object.
// The Rollout object contains callback functions that commits changes in the cluster.
// Prepare is a read-only operation.
func (n *Synchronizer) Prepare(ctx context.Context, source resource.Source) (*Rollout, error) {
	var err error

	rollout := &Rollout{
		Source: source,
	}

	err = source.ApplyDefaults()
	if err != nil {
		return nil, fmt.Errorf("BUG: merge default values into application: %s", err)
	}

	rollout.SynchronizationHash, err = source.Hash(n.config.AivenGeneration)
	if err != nil {
		return nil, fmt.Errorf("BUG: create application hash: %s", err)
	}

	// Skip processing if application didn't change since last synchronization.
	if source.GetStatus().SynchronizationHash == rollout.SynchronizationHash {
		return nil, nil
	}

	err = ensureCorrelationID(source)
	if err != nil {
		return nil, err
	}

	// Prepare for rollout (i.e. use cluster information to generate a configuration object).
	// For this operation, make sure that write operations are disabled.
	opts, err := n.generator.Prepare(ctx, source, readonly.NewClient(n.Client))
	if err != nil {
		return nil, fmt.Errorf("preparing rollout configuration: %w", err)
	}

	rollout.CorrelationID = source.CorrelationID()
	rollout.Options = opts

	return rollout, nil
}

// ClusterOperations generates a set of functions that will perform the rollout in the cluster.
func (n *Synchronizer) ClusterOperations(ctx context.Context, rollout Rollout) []commit {
	funcs := make([]commit, 0)
	deletes := make([]commit, 0)

	// A wrapper to get GroupVersionKind but ensure there's no nils.
	getGroupVersionKind := func(o runtime.Object) schema.GroupVersionKind {
		if o == nil || o.GetObjectKind() == nil {
			return schema.GroupVersionKind{}
		}
		return o.GetObjectKind().GroupVersionKind()
	}

	for _, rop := range rollout.ResourceOperations {
		c := commit{
			groupVersionKind: getGroupVersionKind(rop.Resource),
		}
		n.checkListable(c.groupVersionKind)
		switch rop.Operation {
		case resource.OperationCreateOrUpdate:
			c.fn = updater.CreateOrUpdate(ctx, n.Client, n.scheme, rop.Resource)
		case resource.OperationCreateOrRecreate:
			c.fn = updater.CreateOrRecreate(ctx, n.Client, n.scheme, rop.Resource)
		case resource.OperationCreateIfNotExists:
			c.fn = updater.CreateIfNotExists(ctx, n.Client, rop.Resource)
		case resource.OperationDeleteIfExists:
			c.fn = updater.DeleteIfExists(ctx, n.Client, rop.Resource)
		case resource.AnnotateIfExists:
			c.fn = updater.AnnotateIfExists(ctx, n.Client, n.scheme, rop.Resource)
		default:
			return []commit{
				{
					fn: func() error {
						return fmt.Errorf("BUG: no such operation %s", rop.Operation)
					},
				},
			}
		}

		funcs = append(funcs, c)
	}

	// Delete extraneous resources
	unreferenced, err := n.Unreferenced(ctx, rollout)
	if err != nil {
		deletes = append(deletes, commit{fn: func() error {
			return fmt.Errorf("unable to clean up obsolete resources: %s", err)
		}})
	} else {
		for _, rsrc := range unreferenced {
			deletes = append(deletes, commit{
				groupVersionKind: getGroupVersionKind(rsrc),
				fn:               updater.DeleteIfExists(ctx, n.Client, rsrc.(client.Object)),
			})
		}
	}

	return append(deletes, funcs...)
}

var appsync sync.Mutex

// UpdateResource atomically updates a resource.
// Locks the resource to avoid race conditions.
func (n *Synchronizer) UpdateResource(ctx context.Context, source resource.Source, updateFunc func(resource.Source) error) error {
	appsync.Lock()
	defer appsync.Unlock()

	existing := source.DeepCopyObject().(resource.Source)
	err := n.Get(ctx, client.ObjectKey{Namespace: source.GetNamespace(), Name: source.GetName()}, existing)
	if err != nil {
		return fmt.Errorf("get newest version of %T: %s", existing, err)
	}

	return updateFunc(existing)
}

func (n *Synchronizer) checkListable(kind schema.GroupVersionKind) {
	if !n.isListable(kind) {
		log.Warnf("Resource %s is not listable by any registered Listers", kind.String())
	}
}

func (n *Synchronizer) isListable(objGvk schema.GroupVersionKind) bool {
	if listable, ok := n.listableCache[objGvk]; ok {
		return listable
	}
	for _, lister := range n.listers {
		listGvk, err := apiutil.GVKForObject(lister, n.scheme)
		if err != nil {
			log.Warnf("Error looking up GVK for lister %s: %s", lister, err)
			continue
		}
		if listGvk.GroupVersion() != objGvk.GroupVersion() {
			continue
		}
		if strings.TrimSuffix(listGvk.Kind, "List") == objGvk.Kind {
			n.listableCache[objGvk] = true
			return true
		}
	}
	n.listableCache[objGvk] = false
	return false
}

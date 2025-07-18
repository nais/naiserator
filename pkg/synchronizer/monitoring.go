package synchronizer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nais/liberator/pkg/events"

	deployment "github.com/nais/naiserator/pkg/event"
	"github.com/nais/naiserator/pkg/event/generator"
	"github.com/nais/naiserator/pkg/metrics"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

const (
	RolloutMessageCompleted        = "Rollout has completed"
	RolloutMessageCronJobCompleted = "No support for monitoring CronJobs"
	RolloutMessageJobCompleted     = "Job finished successfully"
	RolloutMessageJobFailed        = "Job has failed"
)

var rolloutMonitorLock sync.Mutex

type RolloutMonitor struct {
	id     uuid.UUID
	cancel context.CancelFunc
}

type completionState struct {
	event              *deployment.Event
	eventReported      bool
	kafkaProduced      bool
	skipKafka          bool
	applicationUpdated bool
}

func (s completionState) produceKafkaMessage() bool {
	return !s.skipKafka && !s.kafkaProduced
}

func (s completionState) saveK8sEvent() bool {
	return !s.eventReported
}

func (s completionState) setSynchronizationState() bool {
	return !s.applicationUpdated && s.eventReported
}

func (n *Synchronizer) produceDeploymentEvent(event *deployment.Event) (int64, error) {
	an, err := anypb.New(event)
	if err != nil {
		return 0, fmt.Errorf("wrap Protobuf.Any: %w", err)
	}
	payload, err := proto.Marshal(an)
	if err != nil {
		return 0, fmt.Errorf("encode Protobuf: %w", err)
	}
	return n.kafka.Produce(payload)
}

func (n *Synchronizer) MonitorRollout(app generator.MonitorSource, logger log.Entry) {
	objectKey := client.ObjectKey{
		Name:      app.GetName(),
		Namespace: app.GetNamespace(),
	}

	// Cancel already running monitor routine if MonitorRollout called again for this particular application.
	n.cancelMonitor(objectKey, nil)

	id := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())
	rolloutMonitorLock.Lock()
	n.rolloutMonitor[objectKey] = RolloutMonitor{
		id:     id,
		cancel: cancel,
	}
	metrics.ResourcesMonitored.Set(float64(len(n.rolloutMonitor)))
	rolloutMonitorLock.Unlock()

	go func() {
		n.monitorRolloutRoutine(ctx, app, logger)
		cancel()
		n.cancelMonitor(objectKey, &id)
	}()
}

func (n *Synchronizer) cancelMonitor(objectKey client.ObjectKey, expected *uuid.UUID) {
	rolloutMonitorLock.Lock()
	defer rolloutMonitorLock.Unlock()

	rollout, ok := n.rolloutMonitor[objectKey]
	if !ok {
		return
	}

	// Avoid race conditions
	if expected != nil && rollout.id.ID() != expected.ID() {
		return
	}

	rollout.cancel()
	delete(n.rolloutMonitor, objectKey)
	metrics.ResourcesMonitored.Set(float64(len(n.rolloutMonitor)))
}

// Monitoring deployments to signal RolloutComplete.
func (n *Synchronizer) monitorRolloutRoutine(ctx context.Context, app generator.MonitorSource, logger log.Entry) {
	logger.Debugf("Monitoring rollout status")

	objectKey := client.ObjectKey{
		Name:      app.GetName(),
		Namespace: app.GetNamespace(),
	}

	completion := completionState{
		// Don't produce Kafka message on certain conditions
		skipKafka: n.kafka == nil || app.SkipDeploymentMessage(),
	}

	for {
		select {
		case <-time.After(n.config.Synchronizer.RolloutCheckInterval):
			switch app.GetObjectKind().GroupVersionKind().Kind {
			case "Naisjob":
				shouldContinue := n.monitorNaisjob(ctx, app, logger, objectKey, completion)
				if shouldContinue {
					continue
				}
				return
			default:
				shouldContinue := n.monitorApplication(ctx, app, logger, objectKey, completion)
				if shouldContinue {
					continue
				}
				return
			}
		case <-ctx.Done():
			logger.Debugf("Monitor rollout: deployment has been redeployed; cancelling monitoring")
			return
		}
	}
}

// monitorNaisjob will return false when the job has completed successfully or failed. We can then stop monitoring this
// deployment. As long as we return true we should keep monitoring the deployment.
func (n *Synchronizer) monitorNaisjob(ctx context.Context, app generator.MonitorSource, logger log.Entry, objectKey client.ObjectKey, completion completionState) bool {
	cronJob := batchv1.CronJob{}
	err := n.Get(ctx, objectKey, &cronJob)
	if err == nil {
		// If we find a cronjob with the same name, it's most probably a Naisjob with schedule, and we can mark the rollout
		// as completed, and finish the monitoring
		err := n.completeRolloutRoutine(ctx, app, logger, completion, events.RolloutComplete, RolloutMessageCronJobCompleted)
		if err != nil {
			logger.Errorf("Monitor rollout: %v", err)
			return true
		}
		return false
	} else if !errors.IsNotFound(err) {
		logger.Errorf("Monitor rollout: failed to query CronJob: %v", err)
	}

	job := &batchv1.Job{}
	err = n.Get(ctx, objectKey, job)
	if err != nil {
		if !errors.IsNotFound(err) {
			logger.Errorf("Monitor rollout: failed to query Job: %v", err)
		}
		return true
	}

	if job.Status.Failed > 0 {
		err = n.completeRolloutRoutine(ctx, app, logger, completion, events.RolloutComplete, RolloutMessageJobFailed)
		if err != nil {
			logger.Errorf("Monitor rollout: store Naisjob sync status: %v", err)
			return true
		}
		return false
	}

	if job.Status.Active == 0 {
		err = n.completeRolloutRoutine(ctx, app, logger, completion, events.RolloutComplete, RolloutMessageJobCompleted)
		if err != nil {
			logger.Errorf("Monitor rollout: %v", err)
			return true
		}
		return false
	}

	return true
}

// monitorApplication will only return false when all pods are successfully up and running. As long as we return true
// we should keep monitoring the deployment.
func (n *Synchronizer) monitorApplication(ctx context.Context, app generator.MonitorSource, logger log.Entry, objectKey client.ObjectKey, completion completionState) bool {
	deploy := &appsv1.Deployment{}
	err := n.Get(ctx, objectKey, deploy)
	if err != nil {
		if !errors.IsNotFound(err) {
			logger.Errorf("Monitor rollout: failed to query Deployment: %v", err)
		}
		return true
	}

	if !applicationDeploymentComplete(deploy) {
		return true
	}

	err = n.completeRolloutRoutine(ctx, app, logger, completion, events.RolloutComplete, RolloutMessageCompleted)
	if err != nil {
		logger.Errorf("Monitor rollout: %v", err)
		return true
	}
	return false
}

func (n *Synchronizer) completeRolloutRoutine(ctx context.Context, app generator.MonitorSource, logger log.Entry, completion completionState, rolloutStatus, rolloutMessage string) error {
	// Deployment event for dev-rapid topic.
	if completion.event == nil {
		logger.Debugf("Monitor rollout: deployment has rolled out completely")
		completion.event = generator.NewDeploymentEvent(app)
		completion.event.RolloutStatus = deployment.RolloutStatus_complete
	}

	// Save a Kubernetes event for this completed deployment.
	// The deployment will be reported as complete when this event is picked up by NAIS deploy.
	if completion.saveK8sEvent() {
		_, err := n.reportEvent(ctx, resource.CreateEvent(app, rolloutStatus, rolloutMessage, "Normal"))
		completion.eventReported = err == nil

		if err != nil {
			return fmt.Errorf("unable to report rollout complete event: %v", err)
		}
	}

	// Send a deployment event to the dev-rapid topic.
	// This is picked up by deployment-event-relays and used as official deployment data.
	if completion.produceKafkaMessage() {
		offset, err := n.produceDeploymentEvent(completion.event)
		completion.kafkaProduced = err == nil
		if err != nil {
			return fmt.Errorf("failed to produce deployment message: %v", err)
		}
		logger.WithFields(log.Fields{
			"kafka_offset": offset,
		}).Infof("Deployment event sent successfully")
	}

	// Set the SynchronizationState field of the application to RolloutComplete.
	// This will prevent the application from being picked up by this function if Naiserator restarts.
	// Only update this field if an event has been persisted to the cluster.
	if completion.setSynchronizationState() {
		err := n.UpdateResource(ctx, app, func(app resource.Source) error {
			app = setSyncStatus(app, rolloutStatus, completion.event)
			return n.Update(ctx, app)
		})

		completion.applicationUpdated = err == nil

		if err != nil {
			return fmt.Errorf("store application sync status: %v", err)
		}
	}

	log.Infof("All systems updated after successful application rollout; terminating monitoring")
	return nil
}

// applicationDeploymentComplete considers a deployment to be complete once all of its desired replicas
// are updated and available, and no old pods are running.
//
// Copied-ish from
// https://github.com/kubernetes/kubernetes/blob/74bcefc8b2bf88a2f5816336999b524cc48cf6c0/pkg/controller/deployment/util/deployment_util.go#L745
func applicationDeploymentComplete(deployment *appsv1.Deployment) bool {
	return deployment.Status.UpdatedReplicas == *(deployment.Spec.Replicas) &&
		deployment.Status.Replicas == *(deployment.Spec.Replicas) &&
		deployment.Status.AvailableReplicas == *(deployment.Spec.Replicas) &&
		deployment.Status.ObservedGeneration >= deployment.Generation
}

func setSyncStatus(app resource.Source, synchronizationState string, event *deployment.Event) resource.Source {
	app.GetStatus().SetSynchronizationStateWithCondition(synchronizationState, "Successfully deployed.")
	app.GetStatus().RolloutCompleteTime = event.GetTimestampAsTime().UnixNano()
	app.GetStatus().DeploymentRolloutStatus = event.RolloutStatus.String()

	metrics.Synchronizations.With(
		prometheus.Labels{
			"kind":   app.GetObjectKind().GroupVersionKind().Kind,
			"status": app.GetStatus().SynchronizationState,
			"team":   app.GetNamespace(),
		},
	).Inc()

	return app
}

package synchronizer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/event"
	"github.com/nais/naiserator/pkg/event/generator"
	"github.com/nais/naiserator/pkg/metrics"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var rolloutMonitorLock sync.Mutex

type RolloutMonitor struct {
	id     uuid.UUID
	cancel context.CancelFunc
}

type completion struct {
	event              *deployment.Event
	eventReported      bool
	kafkaProduced      bool
	applicationUpdated bool
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

func (n *Synchronizer) MonitorRollout(app generator.ImageSource, logger log.Entry) {
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
func (n *Synchronizer) monitorRolloutRoutine(ctx context.Context, app generator.ImageSource, logger log.Entry) {
	logger.Debugf("Monitoring rollout status")

	objectKey := client.ObjectKey{
		Name:      app.GetName(),
		Namespace: app.GetNamespace(),
	}

	completion := completion{}

	// Don't produce Kafka message on certain conditions
	completion.kafkaProduced = n.kafka == nil || app.SkipDeploymentMessage()

	for {
		select {
		case <-time.After(n.config.Synchronizer.RolloutCheckInterval):
			logger.Infof("Kind: %v", app.GetObjectKind().GroupVersionKind().Kind)
			switch app.GetObjectKind().GroupVersionKind().Kind {
			case "Naisjob":
				logger.Info("No monitor for you")
				naisjob := &nais_io_v1.Naisjob{}
				err := n.Get(ctx, objectKey, naisjob)
				if err != nil {
					if !errors.IsNotFound(err) {
						logger.Errorf("Monitor rollout: failed to query Naisjob: %v", err)
					}
					continue
				}
				if naisjob.Spec.Schedule != "" {
					if n.deploymentCompletion(ctx, app, logger, completion) {
						return
					}

					continue
				}
				job := &batchv1.Job{}
				err = n.Get(ctx, objectKey, job)
				if err != nil {
					if !errors.IsNotFound(err) {
						logger.Errorf("Monitor rollout: failed to query Job: %s", err)
					}
					continue
				}

				if job.Status.Failed > 0 {
					// TODO: report failed job
					continue
				}

				if job.Status.Active == 0 {
					if n.deploymentCompletion(ctx, app, logger, completion) {
						return
					}
				}
			default:
				deploy := &appsv1.Deployment{}
				err := n.Get(ctx, objectKey, deploy)

				if err != nil {
					if !errors.IsNotFound(err) {
						logger.Errorf("Monitor rollout: failed to query Deployment: %s", err)
					}
					continue
				}

				if !deploymentComplete(deploy, &deploy.Status) {
					continue
				}

				if n.deploymentCompletion(ctx, app, logger, completion) {
					return
				}
			}

		case <-ctx.Done():
			logger.Debugf("Monitor rollout: application has been redeployed; cancelling monitoring")
			return
		}
	}
}

func (n *Synchronizer) deploymentCompletion(ctx context.Context, app generator.ImageSource, logger log.Entry, completion completion) bool {
	// Deployment event for dev-rapid topic.
	if completion.event == nil {
		logger.Debugf("Monitor rollout: deployment has rolled out completely")
		completion.event = generator.NewDeploymentEvent(app)
		completion.event.RolloutStatus = deployment.RolloutStatus_complete
	}

	// Save a Kubernetes event for this completed deployment.
	// The deployment will be reported as complete when this event is picked up by NAIS deploy.
	if !completion.eventReported {
		_, err := n.reportEvent(ctx, resource.CreateEvent(app, nais_io_v1.EventRolloutComplete, "Deployment rollout has completed", "Normal"))
		completion.eventReported = err == nil
		if err != nil {
			logger.Errorf("Monitor rollout: unable to report rollout complete event: %s", err)
		}
	}

	// Send a deployment event to the dev-rapid topic.
	// This is picked up by deployment-event-relays and used as official deployment data.
	if !completion.kafkaProduced {
		offset, err := n.produceDeploymentEvent(completion.event)
		completion.kafkaProduced = err == nil
		if err == nil {
			logger.WithFields(log.Fields{
				"kafka_offset": offset,
			}).Infof("Deployment event sent successfully")
		} else {
			logger.Errorf("Produce deployment message: %s", err)
		}
	}

	// Set the SynchronizationState field of the application to RolloutComplete.
	// This will prevent the application from being picked up by this function if Naiserator restarts.
	// Only update this field if an event has been persisted to the cluster.
	if !completion.applicationUpdated && completion.eventReported {
		err := n.UpdateResource(ctx, app, func(app resource.Source) error {
			app.GetStatus().SynchronizationState = nais_io_v1.EventRolloutComplete
			app.GetStatus().RolloutCompleteTime = completion.event.GetTimestampAsTime().UnixNano()
			app.GetStatus().DeploymentRolloutStatus = completion.event.RolloutStatus.String()
			app.SetStatusConditions()
			metrics.Synchronizations.WithLabelValues(app.GetObjectKind().GroupVersionKind().Kind, app.GetStatus().SynchronizationState).Inc()
			return n.Update(ctx, app)
		})

		completion.applicationUpdated = err == nil

		if err != nil {
			logger.Errorf("Monitor rollout: store application sync status: %s", err)
		}
	}

	if completion.applicationUpdated && completion.kafkaProduced && completion.eventReported {
		log.Infof("All systems updated after successful application rollout; terminating monitoring")
		return true
	}
	return false
}

// deploymentComplete considers a deployment to be complete once all of its desired replicas
// are updated and available, and no old pods are running.
//
// Copied verbatim from
// https://github.com/kubernetes/kubernetes/blob/74bcefc8b2bf88a2f5816336999b524cc48cf6c0/pkg/controller/deployment/util/deployment_util.go#L745
func deploymentComplete(deployment *appsv1.Deployment, newStatus *appsv1.DeploymentStatus) bool {
	return newStatus.UpdatedReplicas == *(deployment.Spec.Replicas) &&
		newStatus.Replicas == *(deployment.Spec.Replicas) &&
		newStatus.AvailableReplicas == *(deployment.Spec.Replicas) &&
		newStatus.ObservedGeneration >= deployment.Generation
}

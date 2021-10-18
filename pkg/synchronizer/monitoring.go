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
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var rolloutMonitorLock sync.Mutex

type RolloutMonitor struct {
	id     uuid.UUID
	cancel context.CancelFunc
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
	return n.Kafka.Produce(payload)
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
	n.RolloutMonitor[objectKey] = RolloutMonitor{
		id:     id,
		cancel: cancel,
	}
	metrics.ResourcesMonitored.Set(float64(len(n.RolloutMonitor)))
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

	rollout, ok := n.RolloutMonitor[objectKey]
	if !ok {
		return
	}

	// Avoid race conditions
	if expected != nil && rollout.id.ID() != expected.ID() {
		return
	}

	rollout.cancel()
	delete(n.RolloutMonitor, objectKey)
	metrics.ResourcesMonitored.Set(float64(len(n.RolloutMonitor)))
}

// Monitoring deployments to signal RolloutComplete.
func (n *Synchronizer) monitorRolloutRoutine(ctx context.Context, app generator.ImageSource, logger log.Entry) {
	logger.Debugf("Monitoring rollout status")

	objectKey := client.ObjectKey{
		Name:      app.GetName(),
		Namespace: app.GetNamespace(),
	}

	var event *deployment.Event
	var eventReported bool
	var kafkaProduced bool
	var applicationUpdated bool

	// Don't produce Kafka message on certain conditions
	kafkaProduced = n.Kafka == nil || app.SkipDeploymentMessage()

	for {
		select {
		case <-time.After(n.Config.Synchronizer.RolloutCheckInterval):
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

			// Deployment event for dev-rapid topic.
			if event == nil {
				logger.Debugf("Monitor rollout: deployment has rolled out completely")
				event = generator.NewDeploymentEvent(app)
				event.RolloutStatus = deployment.RolloutStatus_complete
			}

			// Save a Kubernetes event for this completed deployment.
			// The deployment will be reported as complete when this event is picked up by NAIS deploy.
			if !eventReported {
				_, err = n.reportEvent(ctx, resource.CreateEvent(app, nais_io_v1.EventRolloutComplete, "Deployment rollout has completed", "Normal"))
				eventReported = err == nil
				if err != nil {
					logger.Errorf("Monitor rollout: unable to report rollout complete event: %s", err)
				}
			}

			// Send a deployment event to the dev-rapid topic.
			// This is picked up by deployment-event-relays and used as official deployment data.
			if !kafkaProduced {
				offset, err := n.produceDeploymentEvent(event)
				kafkaProduced = err == nil
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
			if !applicationUpdated && eventReported {
				err = n.UpdateResource(ctx, app, func(app resource.Source) error {
					app.GetStatus().SynchronizationState = nais_io_v1.EventRolloutComplete
					app.GetStatus().RolloutCompleteTime = event.GetTimestampAsTime().UnixNano()
					app.GetStatus().DeploymentRolloutStatus = event.RolloutStatus.String()
					app.SetStatusConditions()
					metrics.Synchronizations.WithLabelValues(app.GetObjectKind().GroupVersionKind().Kind, app.GetStatus().SynchronizationState).Inc()
					app.SetStatusConditions()
					return n.Update(ctx, app)
				})

				applicationUpdated = err == nil

				if err != nil {
					logger.Errorf("Monitor rollout: store application sync status: %s", err)
				}
			}

			if applicationUpdated && kafkaProduced && eventReported {
				log.Infof("All systems updated after successful application rollout; terminating monitoring")
				return
			}

		case <-ctx.Done():
			logger.Debugf("Monitor rollout: application has been redeployed; cancelling monitoring")
			return
		}
	}
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

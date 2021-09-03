package synchronizer

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
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


type SkatteetatenRolloutMonitor struct {
	id     uuid.UUID
	cancel context.CancelFunc
}

func (n *SkatteetatenApplicationSynchronizer) produceDeploymentEvent(event *deployment.Event) (int64, error) {
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

func (n *SkatteetatenApplicationSynchronizer) MonitorRollout(app *nais_io_v1alpha1.Application, logger log.Entry) {
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
	metrics.ApplicationsMonitored.Set(float64(len(n.RolloutMonitor)))
	rolloutMonitorLock.Unlock()

	go func() {
		n.monitorRolloutRoutine(ctx, app, logger)
		cancel()
		n.cancelMonitor(objectKey, &id)
	}()
}

func (n *SkatteetatenApplicationSynchronizer) cancelMonitor(objectKey client.ObjectKey, expected *uuid.UUID) {
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
	metrics.ApplicationsMonitored.Set(float64(len(n.RolloutMonitor)))
}

// Monitoring deployments to signal RolloutComplete.
func (n *SkatteetatenApplicationSynchronizer) monitorRolloutRoutine(ctx context.Context, app *nais_io_v1alpha1.Application, logger log.Entry) {
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
				event = generator.NewDeploymentEvent(app, app.Spec.Image)
				event.RolloutStatus = deployment.RolloutStatus_complete
			}

			// Save a Kubernetes event for this completed deployment.
			// The deployment will be reported as complete when this event is picked up by NAIS deploy.
			if !eventReported {
				_, err = n.reportEvent(ctx, resource.CreateEvent(app, EventRolloutComplete, "Deployment rollout has completed", "Normal"))
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
				err = n.UpdateApplication(ctx, app, func(app *nais_io_v1alpha1.Application) error {
					app.Status.SynchronizationState = EventRolloutComplete
					app.Status.RolloutCompleteTime = event.GetTimestampAsTime().UnixNano()
					app.SetDeploymentRolloutStatus(event.RolloutStatus.String())
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

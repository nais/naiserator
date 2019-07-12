package kafka

import (
	"time"

	"github.com/google/uuid"

	types "github.com/nais/naiserator/pkg/event"
	log "github.com/sirupsen/logrus"
)

var (
	DeploymentChan chan types.Event

	queueSize = 5
)

func (client *Client) ProducerLoop() {
	DeploymentChan = make(chan types.Event, queueSize)
	go func() {
		for {
			time.Sleep(5 * time.Second)
			uuid, err := uuid.NewRandom()
			id := uuid.String()
			if err != nil {
				id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
			}

			p := types.Platform{Type: types.PlatformType_nais, Variant: "naiserator"}

			a := types.Actor{}

			e := types.Event{CorrelationID: id, Platform: &p, Source: types.System_naiserator, Deployer: &a}
			DeploymentChan <- e
		}
	}()

	for {
		select {
		case msg := <-DeploymentChan:
			log.Info(msg)

		}
	}
}

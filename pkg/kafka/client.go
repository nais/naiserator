package kafka

import (
	"fmt"

	"github.com/Shopify/sarama"
)

type Client struct {
	Producer      sarama.SyncProducer
	ProducerTopic string
}

func NewClient(cfg *Config) (*Client, error) {
	var err error
	client := &Client{}

	producerCfg := sarama.NewConfig()
	producerCfg.ClientID = fmt.Sprintf("%s-producer", cfg.ClientID)
	producerCfg.Net.SASL.Enable = cfg.SASL.Enabled
	producerCfg.Net.SASL.User = cfg.SASL.Username
	producerCfg.Net.SASL.Password = cfg.SASL.Password
	producerCfg.Net.SASL.Handshake = cfg.SASL.Handshake
	producerCfg.Producer.Return.Successes = true
	producerCfg.Producer.RequiredAcks = sarama.WaitForAll
	producerCfg.Producer.Return.Successes = true
	producerCfg.Net.TLS.Enable = cfg.TLS.Enabled

	client.Producer, err = sarama.NewSyncProducer(cfg.Brokers, producerCfg)
	if err != nil {
		return nil, fmt.Errorf("while setting up Kafka producer: %s", err)
	}

	client.ProducerTopic = cfg.Topic

	return client, nil
}

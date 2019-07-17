package kafka

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
)

type SASL struct {
	Enabled   bool
	Handshake bool
	Username  string
	Password  string
}

type TLS struct {
	Enabled  bool
	Insecure bool
}

type Config struct {
	Enabled   bool
	Brokers   string
	Topic     string
	ClientID  string
	GroupID   string
	Verbosity string
	TLS       TLS
	SASL      SASL
}

func DefaultGroupName() string {
	if hostname, err := os.Hostname(); err == nil {
		return fmt.Sprintf("naiserator-%s", hostname)
	}
	return fmt.Sprintf("naiserator-%d", rand.Int())
}

func DefaultConfig() Config {
	defaultGroup := DefaultGroupName()
	return Config{
		Enabled:   false,
		Verbosity: "trace",
		Brokers:   "localhost:9092",
		Topic:     "deploymentEvents",
		ClientID:  defaultGroup,
		GroupID:   defaultGroup,
		SASL: SASL{
			Enabled:   false,
			Handshake: false,
			Username:  os.Getenv("KAFKA_SASL_USERNAME"),
			Password:  os.Getenv("KAFKA_SASL_PASSWORD"),
		},
	}
}

func SetupFlags(cfg *Config) {
	flag.StringVar(&cfg.Brokers, "kafka-brokers", cfg.Brokers, "Comma-separated list of Kafka brokers, HOST:PORT.")
	flag.StringVar(&cfg.Topic, "kafka-topic", cfg.Topic, "Kafka topic for deployment status.")
	flag.StringVar(&cfg.ClientID, "kafka-client-id", cfg.ClientID, "Kafka client ID.")
	flag.StringVar(&cfg.GroupID, "kafka-group-id", cfg.GroupID, "Kafka consumer group ID.")
	flag.StringVar(&cfg.Verbosity, "kafka-log-verbosity", cfg.Verbosity, "Log verbosity for Kafka client.")
	flag.BoolVar(&cfg.SASL.Enabled, "kafka-sasl-enabled", cfg.SASL.Enabled, "Enable SASL authentication.")
	flag.BoolVar(&cfg.SASL.Handshake, "kafka-sasl-handshake", cfg.SASL.Handshake, "Use handshake for SASL authentication.")
	flag.StringVar(&cfg.SASL.Username, "kafka-sasl-username", cfg.SASL.Username, "Username for Kafka authentication.")
	flag.StringVar(&cfg.SASL.Password, "kafka-sasl-password", cfg.SASL.Password, "Password for Kafka authentication.")
	flag.BoolVar(&cfg.TLS.Enabled, "kafka-tls-enabled", cfg.TLS.Enabled, "Use TLS for connecting to Kafka.")
	flag.BoolVar(&cfg.TLS.Insecure, "kafka-tls-insecure", cfg.TLS.Insecure, "Allow insecure Kafka TLS connections.")
	flag.BoolVar(&cfg.Enabled, "kafka-enabled", cfg.Enabled, "Enable connection to kafka")
}

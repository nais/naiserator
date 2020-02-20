package main

import (
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

type Config struct {
	Render bool
}

var cfg = DefaultConfig()

var help = `
validate and/or render NAIS manifests
`

type ExitCode int

const (
	ExitSuccess ExitCode = 0
)

func DefaultConfig() Config {
	return Config{
		Render: false,
	}
}

func init() {
	flag.ErrHelp = fmt.Errorf(help)

	flag.BoolVar(&cfg.Render, "render", cfg.Render, "Render generated NAIS resources to stdout.")

	flag.Parse()

	log.SetOutput(os.Stderr)

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:          true,
		TimestampFormat:        time.RFC3339Nano,
		DisableLevelTruncation: true,
	})
}

func run() (ExitCode, error) {
	var err error

	return ExitSuccess, err
}

func main() {
	code, err := run()
	if err != nil {
		log.Errorf("fatal: %s", err)
	}
	os.Exit(int(code))
}

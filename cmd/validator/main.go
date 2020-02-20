package main

import (
	"fmt"
	"os"
	"time"

	nais_v1 "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Input  string
	Render bool
}

var cfg = DefaultConfig()

var help = `
validate and/or render NAIS manifests
`

type ExitCode int

const (
	ExitSuccess    ExitCode = 0
	ExitInvocation ExitCode = 1
	ExitFile       ExitCode = 2
	ExitFormat     ExitCode = 3
)

func DefaultConfig() Config {
	return Config{
		Render: false,
	}
}

func init() {
	flag.ErrHelp = fmt.Errorf(help)

	flag.StringVar(&cfg.Input, "input", cfg.Input, "nais.yaml input file")
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

	if len(cfg.Input) == 0 {
		return ExitInvocation, fmt.Errorf("must specify input file (try --input)")
	}

	file, err := os.Open(cfg.Input)
	if err != nil {
		return ExitFile, err
	}

	app := &nais_v1.Application{}
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(app)
	if err != nil {
		return ExitFormat, err
	}

	log.Infof("input file '%s' validated successfully", cfg.Input)

	return ExitSuccess, err
}

func main() {
	code, err := run()
	if err != nil {
		log.Errorf("fatal: %s", err)
	}
	os.Exit(int(code))
}

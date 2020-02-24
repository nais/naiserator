package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/ghodss/yaml"
	nais_v1 "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

type Config struct {
	Input  string
	Render bool
}

var cfg = DefaultConfig()

var help = `
Validator verifies that a NAIS manifest is syntactically correct.
It can also, optionally, render Kubernetes resources from an Application resource.
`

type ExitCode int

const (
	ExitSuccess    ExitCode = 0
	ExitInvocation ExitCode = 1
	ExitFile       ExitCode = 2
	ExitFormat     ExitCode = 3
	ExitRender     ExitCode = 4
)

func DefaultConfig() Config {
	return Config{
		Render: false,
	}
}

func init() {
	flag.ErrHelp = fmt.Errorf(help)

	flag.StringVar(&cfg.Input, "input", cfg.Input, "nais.yaml input file")
	flag.BoolVar(&cfg.Render, "render", cfg.Render, "Render generated NAIS resources to stdout (experimental)")

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

	file, err := ioutil.ReadFile(cfg.Input)
	if err != nil {
		return ExitFile, err
	}

	app := &nais_v1.Application{}
	err = yaml.Unmarshal(file, app)
	if err != nil {
		return ExitFormat, err
	}

	log.Infof("input file '%s' validated successfully", cfg.Input)

	if !cfg.Render {
		return ExitSuccess, nil
	}

	err = nais_v1.ApplyDefaults(app)
	if err != nil {
		return ExitRender, err
	}

	opts := resourcecreator.NewResourceOptions()
	operations, err := resourcecreator.Create(app, opts)
	if err != nil {
		return ExitRender, err
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	for _, op := range operations {
		switch op.Operation {
		case resourcecreator.OperationDeleteIfExists:
			continue
		default:
			err = encoder.Encode(op.Resource)
			if err != nil {
				return ExitRender, err
			}
		}
	}

	return ExitSuccess, err
}

func main() {
	code, err := run()
	if err != nil {
		log.Errorf("fatal: %s", err)
	}
	os.Exit(int(code))
}

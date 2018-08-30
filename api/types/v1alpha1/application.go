package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Healthcheck struct {
	Liveness  Probe `json:"liveness"`
	Readiness Probe `json:"readiness"`
}

type Ingress struct {
	Disabled bool `json:"disabled"`
}

type IstioConfig struct {
	Enabled bool `json:"enabled"`
}

type Probe struct {
	Path             string `json:"path"`
	InitialDelay     int    `json:"initialDelay"`
	PeriodSeconds    int    `json:"periodSeconds"`
	FailureThreshold int    `json:"failureThreshold"`
	Timeout          int    `json:"timeout"`
}

type PrometheusConfig struct {
	Enabled bool   `json:"enabled"`
	Port    string `json:"port"`
	Path    string `json:"path"`
}

type Replicas struct {
	Min                    int `json:"min"`
	Max                    int `json:"max"`
	CpuThresholdPercentage int `yaml:"cpuThresholdPercentage"`
}

type ResourceList struct {
	Cpu    string `json:"cpu"`
	Memory string `json:"memory"`
}

type ResourceRequirements struct {
	Limits   ResourceList `json:"limits"`
	Requests ResourceList `json:"requests"`
}

// ApplicationSpec used to be called nais manifest.
type ApplicationSpec struct {
	Healthcheck     Healthcheck          `json:"healthcheck"`
	Image           string               `json:"image"`
	Ingress         Ingress              `json:"ingress"`
	Istio           IstioConfig          `json:"istio"`
	LeaderElection  bool                 `json:"leaderElection"`
	Logformat       string               `json:"logformat"`
	Logtransform    string               `json:"logtransform"`
	Port            int                  `json:"port"`
	PreStopHookPath string               `json:"preStopHookPath"`
	Prometheus      PrometheusConfig     `json:"prometheus"`
	Replicas        Replicas             `json:"replicas"`
	Resources       ResourceRequirements `json:"resources"`
	Secrets         bool                 `json:"secrets"`
	Team            string               `json:"team"`
	WebProxy        bool                 `json:"webproxy"`
}

type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ApplicationSpec `json:"spec"`
}

func (in *Application) GetObjectKind() schema.ObjectKind {
	return in
}

func (in *Application) GetObjectReference() v1.ObjectReference {
	return v1.ObjectReference{
		APIVersion:      "v1alpha1",
		UID:             in.UID,
		Name:            in.Name,
		Kind:            "Application",
		ResourceVersion: in.ResourceVersion,
		Namespace:       in.Namespace,
	}
}

type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Application `json:"items"`
}

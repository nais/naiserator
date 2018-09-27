package v1alpha1

import (
	"strconv"

	hash "github.com/mitchellh/hashstructure"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ApplicationSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Application `json:"items"`
}

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
	CpuThresholdPercentage int `json:"cpuThresholdPercentage"`
}

type ResourceSpec struct {
	Cpu    string `json:"cpu"`
	Memory string `json:"memory"`
}

type ResourceRequirements struct {
	Limits   ResourceSpec `json:"limits"`
	Requests ResourceSpec `json:"requests"`
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

func (in *Application) GetOwnerReference() metav1.OwnerReference {
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         "v1alpha1",
		Kind:               "Application",
		Name:               in.Name,
		UID:                in.UID,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

func (in Application) Hash() (string, error) {
	// struct including the relevant fields for
	// creating a hash of an Application object
	relevantValues := struct {
		TypeMeta metav1.TypeMeta
		AppSpec  ApplicationSpec
		Labels   map[string]string
	}{
		in.TypeMeta,
		in.Spec,
		in.Labels,
	}

	h, err := hash.Hash(relevantValues, nil)
	return strconv.FormatUint(h, 10), err
}

package v1alpha1

// +groupName="nais.io"

import (
	"fmt"
	"strconv"

	hash "github.com/mitchellh/hashstructure"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	LastSyncedHashAnnotation = "nais.io/lastSyncedHash"
	SecretTypeEnv            = "env"
	SecretTypeFiles          = "files"
	DefaultSecretType        = SecretTypeEnv
	DefaultSecretMountPath   = "/var/run/secrets"
)

// Application defines a NAIS application.
//
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Team",type="string",JSONPath=".metadata.labels.team"
// +kubebuilder:resource:path="application",shortName="app"
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ApplicationSpec `json:"spec"`
}

// ApplicationSpec contains the NAIS manifest.
type ApplicationSpec struct {
	AccessPolicy    AccessPolicy         `json:"accessPolicy,omitempty"`
	ConfigMaps      ConfigMaps           `json:"configMaps,omitempty"`
	Env             []EnvVar             `json:"env,omitempty"`
	Image           string               `json:"image"`
	Ingresses       []string             `json:"ingresses,omitempty"`
	LeaderElection  bool                 `json:"leaderElection,omitempty"`
	Liveness        Probe                `json:"liveness,omitempty"`
	Logformat       string               `json:"logformat,omitempty"`
	Logtransform    string               `json:"logtransform,omitempty"`
	Port            int                  `json:"port,omitempty"`
	PreStopHookPath string               `json:"preStopHookPath,omitempty"`
	Prometheus      PrometheusConfig     `json:"prometheus,omitempty"`
	Readiness       Probe                `json:"readiness,omitempty"`
	Replicas        Replicas             `json:"replicas,omitempty"`
	Resources       ResourceRequirements `json:"resources,omitempty"`
	Secrets         []Secret             `json:"secrets,omitempty"`
	SecureLogs      SecureLogs           `json:"secureLogs,omitempty"`
	Service         Service              `json:"service,omitempty"`
	SkipCaBundle    bool                 `json:"skipCaBundle,omitempty"`
	Strategy        Strategy             `json:"strategy,omitempty"`
	Vault           Vault                `json:"vault,omitempty"`
	WebProxy        bool                 `json:"webproxy,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Application `json:"items"`
}

type SecureLogs struct {
	// Whether or not to enable a sidecar container for secure logging.
	Enabled bool `json:"enabled"`
}

// Liveness probe and readiness probe definitions.
type Probe struct {
	Path             string `json:"path"`
	Port             int    `json:"port,omitempty"`
	InitialDelay     int    `json:"initialDelay,omitempty"`
	PeriodSeconds    int    `json:"periodSeconds,omitempty"`
	FailureThreshold int    `json:"failureThreshold,omitempty"`
	Timeout          int    `json:"timeout,omitempty"`
}

type PrometheusConfig struct {
	Enabled bool   `json:"enabled,omitempty"`
	Port    string `json:"port,omitempty"`
	Path    string `json:"path,omitempty"`
}

type Replicas struct {
	// The minimum amount of replicas acceptable for a successful deployment.
	Min int `json:"min,omitempty"`
	// The pod autoscaler will scale deployments on demand until this maximum has been reached.
	Max int `json:"max,omitempty"`
	// Amount of CPU usage before the autoscaler kicks in.
	CpuThresholdPercentage int `json:"cpuThresholdPercentage,omitempty"`
}

type ResourceSpec struct {
	Cpu    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

type ResourceRequirements struct {
	Limits   ResourceSpec `json:"limits,omitempty"`
	Requests ResourceSpec `json:"requests,omitempty"`
}

type ObjectFieldSelector struct {
	FieldPath string `json:"fieldPath"`
}

type EnvVarSource struct {
	FieldRef ObjectFieldSelector `json:"fieldRef"`
}

type EnvVar struct {
	Name      string       `json:"name"`
	Value     string       `json:"value,omitempty"`
	ValueFrom EnvVarSource `json:"valueFrom,omitempty"`
}

type SecretPath struct {
	MountPath string `json:"mountPath"`
	KvPath    string `json:"kvPath"`
}

type Vault struct {
	Enabled bool         `json:"enabled,omitempty"`
	Sidecar bool         `json:"sidecar,omitempty"`
	Mounts  []SecretPath `json:"paths,omitempty"`
}

type ConfigMaps struct {
	Files []string `json:"files,omitempty"`
}

type Strategy struct {
	Type string `json:"type"`
}

type Service struct {
	Port int32 `json:"port"`
}

type AccessPolicyExternalRule struct {
	Host string `json:"host"`
}

type AccessPolicyGressRule struct {
	Application string `json:"application"`
	Namespace   string `json:"namespace,omitempty"`
}

type AccessPolicyInbound struct {
	Rules []AccessPolicyGressRule `json:"rules"`
}

type AccessPolicyOutbound struct {
	Rules    []AccessPolicyGressRule    `json:"rules"`
	External []AccessPolicyExternalRule `json:"external"`
}

type AccessPolicy struct {
	Inbound  AccessPolicyInbound  `json:"inbound,omitempty"`
	Outbound AccessPolicyOutbound `json:"outbound,omitempty"`
}

type Secret struct {
	Name      string `json:"name"`
	Type      string `json:"type,omitempty"`
	MountPath string `json:"mountPath,omitempty"`
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
	return metav1.OwnerReference{
		APIVersion: "v1alpha1",
		Kind:       "Application",
		Name:       in.Name,
		UID:        in.UID,
	}
}

// NilFix initializes all slices from their nil defaults.
//
// This is done in order to workaround the k8s client serializer
// which crashes when these fields are uninitialized.
func (in *Application) NilFix() {
	if in.Spec.Ingresses == nil {
		in.Spec.Ingresses = make([]string, 0)
	}
	if in.Spec.Env == nil {
		in.Spec.Env = make([]EnvVar, 0)
	}
	if in.Spec.Secrets == nil {
		in.Spec.Secrets = make([]Secret, 0)
	}
	if in.Spec.Vault.Mounts == nil {
		in.Spec.Vault.Mounts = make([]SecretPath, 0)
	}
	if in.Spec.ConfigMaps.Files == nil {
		in.Spec.ConfigMaps.Files = make([]string, 0)
	}
	if in.Spec.AccessPolicy.Inbound.Rules == nil {
		in.Spec.AccessPolicy.Inbound.Rules = make([]AccessPolicyGressRule, 0)
	}
	if in.Spec.AccessPolicy.Outbound.Rules == nil {
		in.Spec.AccessPolicy.Outbound.Rules = make([]AccessPolicyGressRule, 0)
	}
}

func (in Application) Hash() (string, error) {
	// struct including the relevant fields for
	// creating a hash of an Application object
	relevantValues := struct {
		AppSpec     ApplicationSpec
		Labels      map[string]string
		ChangeCause string
	}{
		in.Spec,
		in.Labels,
		in.Annotations["kubernetes.io/change-cause"],
	}

	h, err := hash.Hash(relevantValues, nil)
	return strconv.FormatUint(h, 10), err
}

func (in *Application) LastSyncedHash() string {
	a := in.GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	return a[LastSyncedHashAnnotation]
}

func (in *Application) SetLastSyncedHash(hash string) {
	a := in.GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a[LastSyncedHashAnnotation] = hash
	in.SetAnnotations(a)
}

func (in *Application) DefaultSecretPath(base string) SecretPath {
	return SecretPath{
		MountPath: DefaultVaultMountPath,
		KvPath:    fmt.Sprintf("%s/%s/%s", base, in.Name, in.Namespace),
	}
}

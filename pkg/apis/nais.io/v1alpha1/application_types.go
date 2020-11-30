package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	hash "github.com/mitchellh/hashstructure"
	"github.com/nais/naiserator/pkg/event"
	"github.com/nais/naiserator/pkg/naiserator/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	DeploymentCorrelationIDAnnotation = "nais.io/deploymentCorrelationID"
	SkipDeploymentMessageAnnotation   = "nais.io/skipDeploymentMessage"
	DefaultSecretMountPath            = "/var/run/secrets"
	DefaultJwkerMountPath             = "/var/run/secrets/nais.io/jwker"
	DefaultAzureratorMountPath        = "/var/run/secrets/nais.io/azure"
	DefaultKafkaratorMountPath        = "/var/run/secrets/nais.io/kafka"
	DefaultDigdiratorMountPath        = "/var/run/secrets/nais.io/idporten"
)

func GetDefaultMountPath(name string) string {
	return fmt.Sprintf("/var/run/configmaps/%s", name)
}

// Application defines a NAIS application.
//
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Team",type="string",JSONPath=".metadata.labels.team"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.synchronizationState"
// +kubebuilder:resource:path="applications",shortName="app",singular="application"
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec"`
	Status ApplicationStatus `json:"status,omitempty"`
}

// ApplicationSpec contains the NAIS manifest.
type ApplicationSpec struct {
	AccessPolicy    *AccessPolicy         `json:"accessPolicy,omitempty"`
	Azure           *Azure                `json:"azure,omitempty"`
	GCP             *GCP                  `json:"gcp,omitempty"`
	Elastic         *Elastic              `json:"elastic,omitempty"`
	Env             []EnvVar              `json:"env,omitempty"`
	EnvFrom         []EnvFrom             `json:"envFrom,omitempty"`
	FilesFrom       []FilesFrom           `json:"filesFrom,omitempty"`
	Image           string                `json:"image"`
	Ingresses       []string              `json:"ingresses,omitempty"`
	Kafka           *Kafka                `json:"kafka,omitempty"`
	LeaderElection  bool                  `json:"leaderElection,omitempty"`
	Liveness        *Probe                `json:"liveness,omitempty"`
	Logtransform    string                `json:"logtransform,omitempty"`
	Port            int                   `json:"port,omitempty"`
	PreStopHookPath string                `json:"preStopHookPath,omitempty"`
	Prometheus      *PrometheusConfig     `json:"prometheus,omitempty"`
	Readiness       *Probe                `json:"readiness,omitempty"`
	Replicas        *Replicas             `json:"replicas,omitempty"`
	Resources       *ResourceRequirements `json:"resources,omitempty"`
	SecureLogs      *SecureLogs           `json:"secureLogs,omitempty"`
	Service         *Service              `json:"service,omitempty"`
	SkipCaBundle    bool                  `json:"skipCaBundle,omitempty"`
	Startup         *Probe                `json:"startup,omitempty"`
	Strategy        *Strategy             `json:"strategy,omitempty"`
	TokenX          *TokenX               `json:"tokenx,omitempty"`
	Tracing         *Tracing              `json:"tracing,omitempty"`
	Vault           *Vault                `json:"vault,omitempty"`
	WebProxy        bool                  `json:"webproxy,omitempty"`
	IDPorten        *IDPorten             `json:"idporten,omitempty"`

	// +kubebuilder:validation:Enum="";accesslog;accesslog_with_processing_time;accesslog_with_referer_useragent;capnslog;logrus;gokit;redis;glog;simple;influxdb;log15
	Logformat string `json:"logformat,omitempty"`
}

// ApplicationStatus contains different NAIS status properties
type ApplicationStatus struct {
	SynchronizationTime     int64  `json:"synchronizationTime,omitempty"`
	RolloutCompleteTime     int64  `json:"rolloutCompleteTime,omitempty"`
	CorrelationID           string `json:"correlationID,omitempty"`
	DeploymentRolloutStatus string `json:"deploymentRolloutStatus,omitempty"`
	SynchronizationState    string `json:"synchronizationState,omitempty"`
	SynchronizationHash     string `json:"synchronizationHash,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Application `json:"items"`
}

type Azure struct {
	Application *AzureApplication `json:"application"`
}

type Tracing struct {
	// if enabled, a rule allowing egress to app:jaeger will be appended to
	// NetworkPolicy
	Enabled bool `json:"enabled"`
}

type TokenX struct {
	// if enabled, the application will have a jwker secret injected
	Enabled bool `json:"enabled"`
	// if enabled, secrets for TokenX will be mounted as files only, i.e. not as env.
	MountSecretsAsFilesOnly bool `json:"mountSecretsAsFilesOnly,omitempty"`
}

type IDPorten struct {
	Enabled   bool   `json:"enabled"`
	ClientURI string `json:"clientURI,omitempty"`
	// +kubebuilder:validation:Pattern=`^https:\/\/`
	RedirectURI            string   `json:"redirectURI,omitempty"`
	FrontchannelLogoutURI  string   `json:"frontchannelLogoutURI,omitempty"`
	PostLogoutRedirectURIs []string `json:"postLogoutRedirectURIs,omitempty"`
	// +kubebuilder:validation:Minimum=3600
	// +kubebuilder:validation:Maximum=7200
	SessionLifetime *int `json:"sessionLifetime,omitempty"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3600
	AccessTokenLifetime *int `json:"accessTokenLifetime,omitempty"`
}

type AzureApplication struct {
	Enabled   bool     `json:"enabled"`
	ReplyURLs []string `json:"replyURLs,omitempty"`
	// +kubebuilder:validation:Enum=nav.no;trygdeetaten.no
	Tenant string `json:"tenant,omitempty"`
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
	// +kubebuilder:validation:Pattern=^\d+m?$
	Cpu string `json:"cpu,omitempty"`
	// +kubebuilder:validation:Pattern=^\d+[KMG]i$
	Memory string `json:"memory,omitempty"`
}

type ResourceRequirements struct {
	Limits   *ResourceSpec `json:"limits,omitempty"`
	Requests *ResourceSpec `json:"requests,omitempty"`
}

type ObjectFieldSelector struct {
	// +kubebuilder:validation:Enum="";metadata.name;metadata.namespace;metadata.labels;metadata.annotations;spec.nodeName;spec.serviceAccountName;status.hostIP;status.podIP
	FieldPath string `json:"fieldPath"`
}

type EnvVarSource struct {
	FieldRef ObjectFieldSelector `json:"fieldRef"`
}

type CloudStorageBucket struct {
	Name            string `json:"name"`
	CascadingDelete bool   `json:"cascadingDelete,omitempty"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=36500
	RetentionPeriodDays *int `json:"retentionPeriodDays,omitempty"`
	LifecycleCondition *LifecycleCondition `json:"lifecycleCondition,omitempty"`
}

type LifecycleCondition struct {
	Age int `json:"age,omitempty"`
	CreatedBefore string `json:"createdBefore,omitempty"`
	NumNewerVersions int `json:"numNewerVersions,omitempty"`
	WithState string `json:"withState,omitempty"`
}

type CloudSqlInstanceType string

const (
	CloudSqlInstanceTypePostgres11 CloudSqlInstanceType = "POSTGRES_11"
	CloudSqlInstanceTypePostgres12 CloudSqlInstanceType = "POSTGRES_12"
)

type CloudSqlInstanceDiskType string

func (c CloudSqlInstanceDiskType) GoogleType() string {
	return fmt.Sprintf("PD_%s", c)
}

const (
	CloudSqlInstanceDiskTypeSSD CloudSqlInstanceDiskType = "SSD"
	CloudSqlInstanceDiskTypeHDD CloudSqlInstanceDiskType = "HDD"
)

type CloudSqlDatabase struct {
	// +kubebuilder:validation:Required
	Name         string `json:"name"`
	EnvVarPrefix string `json:"envVarPrefix,omitempty"`
}

type CloudSqlInstance struct {
	// +kubebuilder:validation:Enum=POSTGRES_11;POSTGRES_12
	// +kubebuilder:validation:Required
	Type CloudSqlInstanceType `json:"type"`
	Name string               `json:"name,omitempty"`
	// +kubebuilder:validation:Pattern="db-.+"
	Tier string `json:"tier,omitempty"`
	// +kubebuilder:validation:Enum=SSD;HDD
	DiskType         CloudSqlInstanceDiskType `json:"diskType,omitempty"`
	HighAvailability bool                     `json:"highAvailability,omitempty"`
	// +kubebuilder:validation:Minimum=10
	DiskSize       int  `json:"diskSize,omitempty"`
	DiskAutoresize bool `json:"diskAutoresize,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=23
	AutoBackupHour *int         `json:"autoBackupHour,omitempty"` // must use pointer here to be able to distinguish between no/zero value and value 0 from user.
	Maintenance    *Maintenance `json:"maintenance,omitempty"`
	// +kubebuilder:validation:Required
	Databases       []CloudSqlDatabase `json:"databases,omitempty"`
	CascadingDelete bool               `json:"cascadingDelete,omitempty"`
}

type Maintenance struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=7
	Day int `json:"day,omitempty"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=23
	Hour *int `json:"hour,omitempty"` // must use pointer here to be able to distinguish between no value and value 0 from user.
}

type Elastic struct {
	Instance string `json:"instance"`
}

type GCP struct {
	Buckets      []CloudStorageBucket `json:"buckets,omitempty"`
	SqlInstances []CloudSqlInstance   `json:"sqlInstances,omitempty"`
}

type EnvVar struct {
	Name      string        `json:"name"`
	Value     string        `json:"value,omitempty"`
	ValueFrom *EnvVarSource `json:"valueFrom,omitempty"`
}

type EnvFrom struct {
	ConfigMap string `json:"configmap,omitempty"`
	Secret    string `json:"secret,omitempty"`
}

type FilesFrom struct {
	ConfigMap string `json:"configmap,omitempty"`
	Secret    string `json:"secret,omitempty"`
	MountPath string `json:"mountPath,omitempty"`
}

type SecretPath struct {
	MountPath string `json:"mountPath"`
	KvPath    string `json:"kvPath"`
	// +kubebuilder:validation:Enum=flatten;json;yaml;env;properties;""
	Format string `json:"format,omitempty"`
}

type Vault struct {
	Enabled bool         `json:"enabled,omitempty"`
	Sidecar bool         `json:"sidecar,omitempty"`
	Paths   []SecretPath `json:"paths,omitempty"`
}

type Strategy struct {
	// +kubebuilder:validation:Enum=Recreate;RollingUpdate
	Type string `json:"type"`
}

type Service struct {
	// +kubebuilder:validation:Enum=http;redis;tcp
	Protocol string `json:"protocol,omitempty"`
	Port     int32  `json:"port"`
}

type AccessPolicyPortRule struct {
	Name string `json:"name"`
	Port uint32 `json:"port"`
	// +kubebuilder:validation:Enum=HTTP;HTTPS;GRPC;HTTP2;MONGO;TCP;TLS
	Protocol string `json:"protocol"`
}

type AccessPolicyExternalRule struct {
	Host  string                 `json:"host"`
	Ports []AccessPolicyPortRule `json:"ports,omitempty"`
}

type AccessPolicyRule struct {
	Application string `json:"application"`
	Namespace   string `json:"namespace,omitempty"`
	Cluster     string `json:"cluster,omitempty"`
}

type AccessPolicyInbound struct {
	Rules []AccessPolicyRule `json:"rules"`
}

type AccessPolicyOutbound struct {
	Rules    []AccessPolicyRule         `json:"rules,omitempty"`
	External []AccessPolicyExternalRule `json:"external,omitempty"`
}

type AccessPolicy struct {
	Inbound  *AccessPolicyInbound  `json:"inbound,omitempty"`
	Outbound *AccessPolicyOutbound `json:"outbound,omitempty"`
}

type Kafka struct {
	Pool string `json:"pool"`
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

func (in Application) Hash() (string, error) {
	// struct including the relevant fields for
	// creating a hash of an Application object
	var changeCause string
	if in.Annotations != nil {
		changeCause = in.Annotations["kubernetes.io/change-cause"]
	}
	relevantValues := struct {
		AppSpec     ApplicationSpec
		Labels      map[string]string
		ChangeCause string
	}{
		in.Spec,
		in.Labels,
		changeCause,
	}

	marshalled, err := json.Marshal(relevantValues)
	if err != nil {
		return "", err
	}
	h, err := hash.Hash(marshalled, nil)
	return fmt.Sprintf("%x", h), err
}

func (in *Application) LogFields() log.Fields {
	return log.Fields{
		"namespace":       in.GetNamespace(),
		"resourceversion": in.GetResourceVersion(),
		"application":     in.GetName(),
		"correlation_id":  in.Status.CorrelationID,
	}
}

func (in Application) Cluster() string {
	return viper.GetString(config.ClusterName)
}

// If the application was deployed with a correlation ID annotation, return this value.
// Otherwise, generate a random UUID.
func (in *Application) NextCorrelationID() (string, error) {
	var correlationID string

	if in.Annotations != nil {
		correlationID = in.Annotations[DeploymentCorrelationIDAnnotation]
	}

	if len(correlationID) == 0 {
		id, err := uuid.NewRandom()
		if err != nil {
			return correlationID, fmt.Errorf("BUG: generate deployment correlation ID: %s", err)
		}
		correlationID = id.String()
	}

	return correlationID, nil
}

func (in *Application) SetCorrelationID(id string) {
	in.Status.CorrelationID = id
}

func (in *Application) SetDeploymentRolloutStatus(rolloutStatus deployment.RolloutStatus) {
	in.Status.DeploymentRolloutStatus = rolloutStatus.String()
}

func (in *Application) DefaultSecretPath(base string) SecretPath {
	return SecretPath{
		MountPath: DefaultVaultMountPath,
		KvPath:    fmt.Sprintf("%s/%s/%s", base, in.Name, in.Namespace),
	}
}

func (in *Application) SkipDeploymentMessage() bool {
	if in.Annotations == nil {
		return false
	}
	skip, _ := strconv.ParseBool(in.Annotations[SkipDeploymentMessageAnnotation])
	return skip
}

func (in AccessPolicyRule) MatchesCluster(clusterName string) bool {
	if len(in.Cluster) > 0 && in.Cluster != clusterName {
		return false
	}
	return true
}

func (in *Application) ClientID() string {
	return fmt.Sprintf("%s:%s:%s", in.Cluster(), in.ObjectMeta.Namespace, in.ObjectMeta.Name)
}

func (in *Application) AddAccessPolicyExternalHostsAsStrings(hosts []string) {
	externalRules := make([]AccessPolicyExternalRule, len(hosts))
	for _, host := range hosts {
		externalRules = append(externalRules, AccessPolicyExternalRule{Host: host})
	}

	in.AddAccessPolicyExternalHosts(externalRules)
}

func (in *Application) AddAccessPolicyExternalHosts(hosts []AccessPolicyExternalRule) {
	var empty struct{}
	seen := map[string]struct{}{}
	rules := make([]AccessPolicyExternalRule, 0)

	for _, rule := range in.Spec.AccessPolicy.Outbound.External {
		seen[rule.Host] = empty
	}

	for _, externalRule := range hosts {
		if len(externalRule.Host) == 0 {
			continue
		}
		if _, found := seen[externalRule.Host]; !found {
			seen[externalRule.Host] = empty
			rules = append(rules, externalRule)
		}
	}

	in.Spec.AccessPolicy.Outbound.External = append(in.Spec.AccessPolicy.Outbound.External, rules...)
}

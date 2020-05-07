package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	EventRolloutComplete       = "RolloutComplete"
	EventFailedPrepare         = "FailedPrepare"
	EventFailedSynchronization = "FailedSynchronization"
)

type JwkerSpec struct {
	AccessPolicy *AccessPolicy `json:"accessPolicy"`
	SecretName   string        `json:"secretName"`
}
type AccessPolicy struct {
	Inbound  *AccessPolicyInbound  `json:"inbound,omitempty"`
	Outbound *AccessPolicyOutbound `json:"outbound,omitempty"`
}
type AccessPolicyOutbound struct {
	Rules []AccessPolicyRule `json:"rules,omitempty"`
}
type AccessPolicyInbound struct {
	Rules []AccessPolicyRule `json:"rules,omitempty"`
}
type AccessPolicyRule struct {
	Application string `json:"application"`
	Namespace   string `json:"namespace,omitempty"`
	Cluster     string `json:"cluster,omitempty"`
}

// JwkerStatus defines the observed state of Jwker
type JwkerStatus struct {
	SynchronizationTime  int64  `json:"synchronizationTime,omitempty"`
	SynchronizationState string `json:"synchronizationState,omitempty"`
	SynchronizationHash  string `json:"synchronizationHash,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Jwker is the Schema for the jwkers API
type Jwker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JwkerSpec   `json:"spec,omitempty"`
	Status JwkerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// JwkerList contains a list of Jwker
type JwkerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Jwker `json:"items"`
}

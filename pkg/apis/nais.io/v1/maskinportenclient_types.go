package v1

// +groupName="nais.io"

import (
	"github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MaskinportenClient is the Schema for the MaskinportenClient API
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MaskinportenClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MaskinportenClientSpec   `json:"spec,omitempty"`
	Status MaskinportenClientStatus `json:"status,omitempty"`
}

// MaskinportenClientList contains a list of MaskinportenClient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MaskinportenClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MaskinportenClient `json:"items"`
}

// MaskinportenClientSpec defines the desired state of MaskinportenClient
type MaskinportenClientSpec struct {
	// Scopes is a list of valid scopes that the client can request tokens for
	Scopes []v1alpha1.MaskinportenScope `json:"scopes"`
	// SecretName is the name of the resulting Secret resource to be created
	SecretName string `json:"secretName"`
}

// MaskinportenClientStatus defines the observed state of MaskinportenClient
type MaskinportenClientStatus struct {
	// SynchronizationState denotes the last known state of the MaskinportenClient during synchronization
	SynchronizationState string `json:"synchronizationState,omitempty"`
	// SynchronizationTime is the last time the Status subresource was updated
	SynchronizationTime *metav1.Time `json:"synchronizationTime,omitempty"`
	// SynchronizationHash is the hash of the MaskinportenClient object
	SynchronizationHash string `json:"synchronizationHash,omitempty"`
	// ClientID is the corresponding client ID for this client at Digdir
	ClientID string `json:"clientID,omitempty"`
	// CorrelationID is the ID referencing the processing transaction last performed on this resource
	CorrelationID string `json:"correlationID,omitempty"`
	// KeyIDs is the list of key IDs for valid JWKs registered for the client at Digdir
	KeyIDs []string `json:"keyIDs,omitempty"`
}

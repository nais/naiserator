package v1

// +groupName="nais.io"

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IDPortenClient is the Schema for the IDPortenClients API
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type IDPortenClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IDPortenClientSpec   `json:"spec,omitempty"`
	Status IDPortenClientStatus `json:"status,omitempty"`
}

// IDPortenClientList contains a list of IDPortenClient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type IDPortenClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IDPortenClient `json:"items"`
}

// IDPortenClientSpec defines the desired state of IDPortenClient
type IDPortenClientSpec struct {
	// ClientName is the name of the client registered at DigDir
	ClientName string `json:"clientName"`
	// ClientURI is the URL to the client to be used at DigDir when displaying a 'back' button or on errors
	ClientURI string `json:"clientURI"`
	// RedirectURIs is a list of redirect URIs to be registered at DigDir
	RedirectURIs []string `json:"redirectURIs"`
	// SecretName is the name of the resulting Secret resource to be created
	SecretName string `json:"secretName"`
	// FrontchannelLogoutURI is the URL that ID-porten sends a requests to whenever a logout is triggered by another application using the same session
	FrontchannelLogoutURI string `json:"frontchannelLogoutURI,omitempty"`
	// PostLogoutRedirectURI is a list of valid URIs that ID-porten may redirect to after logout
	PostLogoutRedirectURIs []string `json:"postLogoutRedirectURIs"`
	// RefreshTokenLifetime is the lifetime in seconds for the issued refresh token from ID-porten
	RefreshTokenLifetime int `json:"refreshTokenLifetime"`
}

// IDPortenClientStatus defines the observed state of IDPortenClient
type IDPortenClientStatus struct {
	// ClientID is the ID-porten client ID
	ClientID string `json:"clientID"`
	// Timestamp is the last time the Status subresource was updated
	Timestamp metav1.Time `json:"timestamp,omitempty"`
	// ProvisionHash is the hash of the IDPortenClient object
	ProvisionHash string `json:"provisionHash,omitempty"`
	// CorrelationID is the ID referencing the processing transaction last performed on this resource
	CorrelationID string `json:"correlationID"`
	// KeyIDs is the list of key IDs for valid JWKs registered for the client at ID-porten
	KeyIDs []string `json:"keyIDs"`
}

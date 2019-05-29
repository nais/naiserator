package v1alpha3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []VirtualService `json:"items"`
}

type PortSelector struct {
	Number uint32 `json:"number"`
}

type Destination struct {
	Host   string       `json:"host"`
	Subset string       `json:"subset"`
	Port   PortSelector `json:"port"`
}

type HTTPRouteDestination struct {
	Destination Destination `json:"destination"`
	Weight      int32       `json:"weight"`
}

type HTTPRoute struct {
	Route []HTTPRouteDestination `json:"route"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VirtualServiceSpec `json:"spec"`
}

type VirtualServiceSpec struct {
	Gateways []string    `json:"gateways"`
	Hosts    []string    `json:"hosts"`
	HTTP     []HTTPRoute `json:"http"`
}

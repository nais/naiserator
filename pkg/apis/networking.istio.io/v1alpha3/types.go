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

type StringMatch struct {
	Exact  string `json:"exact,omitempty"`
	Prefix string `json:"prefix,omitempty"`
	Regex  string `json:"regex,omitempty"`
}

type HTTPMatchRequest struct {
	URI StringMatch `json:"uri"`
}

type HTTPRouteDestination struct {
	Destination Destination `json:"destination"`
	Weight      int32       `json:"weight"`
}

type HTTPRoute struct {
	Match []HTTPMatchRequest     `json:"match,omitempty"`
	Route []HTTPRouteDestination `json:"route"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// documentation: https://istio.io/docs/reference/config/networking/virtual-service/
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ServiceEntryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ServiceEntry `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//
// documentation: https://istio.io/docs/reference/config/networking/v1alpha3/service-entry
type ServiceEntry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ServiceEntrySpec `json:"spec"`
}

type ServiceEntrySpec struct {
	Addresses  []string `json:"addresses,omitempty"`
	Hosts      []string `json:"hosts"`
	Location   string   `json:"location,omitempty"`
	Resolution string   `json:"resolution"`
	Ports      []Port   `json:"ports"`
}

type Port struct {
	Number   uint32 `json:"number"`
	Protocol string `json:"protocol"`
	Name     string `json:"name,omitempty"`
}

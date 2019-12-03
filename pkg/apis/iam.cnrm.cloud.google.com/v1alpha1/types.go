package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type IAMServiceAccount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec IAMServiceAccountSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type IAMServiceAccountList struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items []IAMServiceAccount `json:"items"`
}

type IAMServiceAccountSpec struct {
	DisplayName string `json:"displayName"`
}

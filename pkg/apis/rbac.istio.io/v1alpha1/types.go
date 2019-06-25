package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ServiceRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ServiceRoleSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ServiceRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ServiceRole `json:"items"`
}

type ServiceRoleSpec struct {
	Rules []*AccessRule `json:"rules"`
}

type AccessRule struct {
	Services []string `json:"services"`
	Methods  []string `json:"methods"`
	Paths 	 []string `json:"Paths"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ServiceRoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ServiceRoleBindingSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ServiceRoleBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ServiceRoleBinding `json:"items"`
}

type ServiceRoleBindingSpec struct {
	Subjects []*Subject `json:"subjects"`
	RoleRef  *RoleRef   `json:"roleRef"`
}

type Subject struct {
	User string `json:"user"`
}

type RoleRef struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

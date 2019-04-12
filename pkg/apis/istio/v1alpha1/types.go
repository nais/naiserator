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

type ServiceRoleSpec struct {
	Rules []*AccessRule
}

type AccessRule struct {
	Services []string
	Methods  []string
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ServiceRoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ServiceRoleBindingSpec `json:"spec"`
}

type ServiceRoleBindingSpec struct {
	Subjects []*Subject
	RoleRef *RoleRef
}

type Subject struct {
	User string
}

type RoleRef struct {
	Kind string
	Name string
}
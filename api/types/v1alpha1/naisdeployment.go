package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type NaisDeploymentSpec struct {
	A string `json:"a"`
}

type NaisDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec NaisDeploymentSpec `json:"spec"`
}

type NaisDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []NaisDeployment `json:"items"`
}

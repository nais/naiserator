package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SQLInstanceSpec struct {
	DatabaseVersion string              `json:"databaseVersion"`
	Region          string              `json:"region"`
	Settings        SQLInstanceSettings `json:"settings"`
}

type SQLInstanceSettings struct {
	AvailabilityType    string                         `json:"availabilityType"`
	BackupConfiguration SQLInstanceBackupConfiguration `json:"backupConfiguration"`
	DiskAutoresize      bool                           `json:"diskAutoresize"`
	DiskSize            int                            `json:"diskSize"`
	DiskType            string                         `json:"diskType"`
	Tier                string                         `json:"tier"`
}

type SQLInstanceBackupConfiguration struct {
	Enabled   bool   `json:"enabled"`
	StartTime string `json:"startTime"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SQLInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              SQLInstanceSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SQLInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SQLInstance `json:"items"`
}

type InstanceRef struct {
	Name string `json:"name"`
}
type SQLDatabaseSpec struct {
	InstanceRef InstanceRef `json:"instanceRef"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SQLDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              SQLDatabaseSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SQLDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SQLDatabase `json:"items"`
}

type SecretRef struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type SqlUserPasswordSecretKeyRef struct {
	SecretKeyRef SecretRef `json:"secretKeyRef"`
}

type SqlUserPasswordValue struct {
	ValueFrom SqlUserPasswordSecretKeyRef `json:"valueFrom"`
}

type SQLUserSpec struct {
	InstanceRef InstanceRef          `json:"instanceRef"`
	Host        string               `json:"host"`
	Password    SqlUserPasswordValue `json:"password"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SQLUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              SQLUserSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SQLUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SQLUser `json:"items"`
}

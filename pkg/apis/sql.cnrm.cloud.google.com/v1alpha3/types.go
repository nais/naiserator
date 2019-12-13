package v1alpha3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SqlInstanceSpec struct {
	DatabaseVersion string              `json:"databaseVersion"`
	Region          string              `json:"region"`
	Settings        SqlInstanceSettings `json:"settings"`
}

type SqlInstanceSettings struct {
	AvailabilityType    string                         `json:"availabilityType"`
	BackupConfiguration SqlInstanceBackupConfiguration `json:"backupConfiguration"`
	DiskAutoResize      bool                           `json:"diskAutoResize"`
	DiskSize            int                            `json:"diskSize"`
	DiskType            string                         `json:"diskType"`
	Tier                string                         `json:"tier"`
}

type SqlInstanceBackupConfiguration struct {
	Enabled   bool   `json:"enabled"`
	StartTime string `json:"startTime"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SqlInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty'`
	Spec              SqlInstanceSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SqlInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SqlInstance `json:"items"`
}

type InstanceRef struct {
	Name string `json:"name"`
}
type SqlDatabaseSpec struct {
	InstanceRef InstanceRef `json:"instanceRef"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SqlDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty'`
	Spec              SqlDatabaseSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SqlDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SqlDatabase `json:"items"`
}

type SqlUserSpec struct {
	InstanceRef InstanceRef `json:"instanceRef"`
	Host        string      `json:"host"`
	Password    string      `json:"password"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SqlUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty'`
	Spec              SqlUserSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SqlUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SqlUserList `json:"items"`
}

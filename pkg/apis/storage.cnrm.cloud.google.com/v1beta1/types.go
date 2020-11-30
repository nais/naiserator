package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type StorageBucket struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              StorageBucketSpec `json:"spec"`
}

type StorageBucketSpec struct {
	Location        string           `json:"location"`
	RetentionPolicy *RetentionPolicy `json:"retentionPolicy,omitempty"`
	LifecycleRules   []LifecycleRules  `json:"lifecycleRule,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type StorageBucketList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StorageBucket `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type StorageBucketAccessControl struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              StorageBucketAccessControlSpec `json:"spec"`
}

type StorageBucketAccessControlSpec struct {
	BucketRef BucketRef `json:"bucketRef"`
	Entity    string    `json:"entity"`
	Role      string    `json:"role"`
}

type BucketRef struct {
	Name string `json:"name"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type StorageBucketAccessControlList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StorageBucketAccessControl `json:"items"`
}

type RetentionPolicy struct {
	RetentionPeriod int `json:"retentionPeriod,omitempty"`
}

type LifecycleRules struct {
	Action    Action  `json:"action"`
	Condition Condition `json:"condition"`
}

type Action struct {
	Type string `json:"type,omitempty"`
}

type Condition struct {
	Age              int    `json:"age,omitempty"`
	CreatedBefore    string `json:"createdBefore,omitempty"`
	NumNewerVersions int    `json:"numNewerVersions,omitempty"`
	WithState        string `json:"withState,omitempty"`
}

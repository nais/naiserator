package resource

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Source interface {
	metav1.Object
	CreateObjectMeta() metav1.ObjectMeta
	CreateAppNamespaceHash() string
}

type Ast struct {
	Operations Operations

	// For podSpec
	Annotations  map[string]string
	Containers   []v1.Container
	Envs         []v1.EnvVar
	Labels       map[string]string
	Volumes      []v1.Volume
	VolumeMounts []v1.VolumeMount
}

func NewAst() *Ast {
	return &Ast{
		Operations: []Operation{},

		Annotations:  map[string]string{},
		Containers:   []v1.Container{},
		Envs:         []v1.EnvVar{},
		Labels:       map[string]string{},
		Volumes:      []v1.Volume{},
		VolumeMounts: []v1.VolumeMount{},
	}
}

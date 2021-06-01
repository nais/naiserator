package resource

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (ast *Ast) AppendOperation(operationType OperationType, resource runtime.Object) {
	ast.Operations = append(ast.Operations, Operation{
		Operation: operationType,
		Resource: resource,
	})
}

type Ast struct {
	Operations Operations

	// For podSpec
	Annotations    map[string]string
	Containers     []v1.Container
	Env            []v1.EnvVar
	EnvFrom        []v1.EnvFromSource
	InitContainers []v1.Container
	Labels         map[string]string
	Volumes        []v1.Volume
	VolumeMounts   []v1.VolumeMount
}

func NewAst() *Ast {
	return &Ast{
		Operations: []Operation{},

		Annotations:  map[string]string{},
		Containers:   []v1.Container{},
		Env:          []v1.EnvVar{},
		Labels:       map[string]string{},
		Volumes:      []v1.Volume{},
		VolumeMounts: []v1.VolumeMount{},
	}
}

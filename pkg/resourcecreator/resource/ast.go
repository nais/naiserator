package resource

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (ast *Ast) AppendOperation(operationType OperationType, resource runtime.Object) {
	ast.Operations = append(ast.Operations, Operation{
		Operation: operationType,
		Resource:  resource,
	})
}

type Ast struct {
	Operations Operations

	// For podSpec
	Annotations    map[string]string
	Containers     []corev1.Container
	Env            []corev1.EnvVar
	EnvFrom        []corev1.EnvFromSource
	InitContainers []corev1.Container
	JobSpec        batchv1.JobSpec
	Labels         map[string]string
	Volumes        []corev1.Volume
	VolumeMounts   []corev1.VolumeMount
}

func NewAst() *Ast {
	return &Ast{
		Operations: []Operation{},

		Annotations:  map[string]string{},
		Containers:   []corev1.Container{},
		Env:          []corev1.EnvVar{},
		Labels:       map[string]string{},
		Volumes:      []corev1.Volume{},
		VolumeMounts: []corev1.VolumeMount{},
	}
}

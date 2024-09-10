package resource

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (ast *Ast) AppendOperation(operationType OperationType, resource client.Object) {
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

// Use AppendEnv for adding environment variables that depend on other environment variables, or to ensure that they can not be overridden by the user.
func (a *Ast) AppendEnv(vars ...corev1.EnvVar) {
	a.Env = append(a.Env, vars...)
}

// Use PrependEnv for adding environment variables that can be referenced by other environment variables or allowed to be overriden by the user.
func (a *Ast) PrependEnv(vars ...corev1.EnvVar) {
	a.Env = append(vars, a.Env...)
}

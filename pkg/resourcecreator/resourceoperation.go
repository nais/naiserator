package resourcecreator

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// Operation defines what should be done with a resource.
type Operation int

const (
	OperationCreateOrUpdate Operation = iota
	OperationCreateOrRecreate
	OperationDeleteIfExists
)

// ResourceOperation is the combination of a Kubernetes resource and what operation to perform on it.
type ResourceOperation struct {
	Resource  runtime.Object
	Operation Operation
}


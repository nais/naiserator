package resourcecreator

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// Operation defines what should be done with a resource.
type Operation int

const (
	OperationCreateOrUpdate Operation = iota
	OperationCreateOrRecreate
	OperationCreateIfNotExists
	OperationDeleteIfExists
)

// ResourceOperation is the combination of a Kubernetes resource and what operation to perform on it.
type ResourceOperation struct {
	Resource  runtime.Object
	Operation Operation
}

type ResourceOperations []ResourceOperation

// Return a new slice of ResourceOperations where only one type of operation matches.
func (r *ResourceOperations) Extract(operation Operation) ResourceOperations {
	results := make(ResourceOperations, 0)
	for _, rop := range *r {
		if rop.Operation == operation {
			results = append(results, rop)
		}
	}
	return results
}

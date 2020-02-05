package resourcecreator

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// Operation defines what should be done with a resource.
type Operation string

const (
	OperationCreateOrUpdate    Operation = `CreateOrUpdate`
	OperationCreateOrRecreate            = `CreateOrRecreate`
	OperationCreateIfNotExists           = `CreateIfNotExists`
	OperationDeleteIfExists              = `DeleteIfExists`
)

// ResourceOperation is the combination of a Kubernetes resource and what operation to perform on it.
type ResourceOperation struct {
	Resource  runtime.Object `json:"resource"`
	Operation Operation      `json:"operation"`
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

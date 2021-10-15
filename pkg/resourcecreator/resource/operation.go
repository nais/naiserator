package resource

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OperationType defines what should be done with a resource.
type OperationType string

const (
	OperationCreateOrUpdate    OperationType = `CreateOrUpdate`
	OperationCreateOrRecreate  OperationType = `CreateOrRecreate`
	OperationCreateIfNotExists OperationType = `CreateIfNotExists`
	OperationDeleteIfExists    OperationType = `CreateDeleteIfExists`
)

// Operation is the combination of a Kubernetes resource and what operation to perform on it.
type Operation struct {
	Resource  client.Object `json:"resource"`
	Operation OperationType `json:"operation"`
}

type Operations []Operation

// Extract return a new slice of Operations where only one type of operation matches.
func (r *Operations) Extract(operation OperationType) Operations {
	results := make(Operations, 0)
	for _, rop := range *r {
		if rop.Operation == operation {
			results = append(results, rop)
		}
	}
	return results
}

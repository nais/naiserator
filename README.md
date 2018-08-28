# naiserator

Naiserator is an operator that abstracts a number of different Kubernetes resources in order to simplify deployment of applications at NAV.

## Getting started

```
minikube start
kubectl apply -f api/types/v1alpha1/application.yaml
kubectl apply -f examples/nais_example.yaml
go run *.go --kubeconfig=<path-to-kubeconfig>
```

## Differences from previous nais.yaml

* The `redis` field has been removed ([#6](issues/6))
* The `alerts` field has been removed ([#7](issues/6))
* The `fasitResources` field has been removed ([#9](issues/6))

# naiserator

Naiserator is an operator that abstracts a number of different Kubernetes resources in order to simplify deployment of applications at NAV.

## Getting started

```
minikube start
kubectl apply -f api/types/v1alpha1/application.yaml
kubectl apply -f examples/nais_example.yaml
go run *.go --kubeconfig=<path-to-kubeconfig>
```

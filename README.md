# Naiserator

Naiserator is a Kubernetes operator which creates many different resources based on a high level abstraction,
in order to simplify application deployment at NAV.
Naiserator supercedes [naisd](https://nais.io).

When an `Application` resource is created in Kubernetes (see
[example application](api/types/v1alpha1/application.yaml)),
Naiserator will generate several resources that work together to form a complete deployment:
  * `Deployment` that runs a specified number of application instances,
  * `Service` which points to the application endpoint,
  * `Ingress` adding TLS termination and virtualhost support,
  * `Horizontal pod autoscaler` for automatic application scaling,
  * `ConfigMaps`, one for environment variables, and another for files.
  
These resources will remain in Kubernetes until the `Application` resource is deleted.
  
## Prerequisites

* For deployment, [Kubernetes](https://kubernetes.io/) v1.11.0 or later
* For development, the [Go](https://golang.org/dl/) programming language, version 1.11 or later

## Installation

Production builds can, in the future, be installed by:
```
kubectl apply -f kubernetes/naiserator.yml
```

## Development

[Go modules](https://github.com/golang/go/wiki/Modules)
are used for dependency tracking. Make sure you do `export GO111MODULE=on` before running any Go commands.
It is no longer needed to have the project checked out in your `$GOPATH`.

```
minikube start
kubectl apply -f api/types/v1alpha1/application.yaml
kubectl apply -f examples/nais_example.yaml
make
cmd/naiserator/naiserator --kubeconfig=<path-to-kubeconfig>
```

## Differences from previous nais.yaml

* The `redis` field has been removed ([#6][i6])
* The `alerts` field has been removed ([#7][i7])
* The `fasitResources` field has been removed ([#9][i9])

[i6]: https://github.com/nais/naiserator/issues/6
[i7]: https://github.com/nais/naiserator/issues/7
[i9]: https://github.com/nais/naiserator/issues/9

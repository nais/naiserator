# Naiserator

[![CircleCI](https://circleci.com/gh/nais/naiserator/tree/master.svg?style=svg)](https://circleci.com/gh/nais/naiserator/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/nais/naiserator)](https://goreportcard.com/report/github.com/nais/naiserator)

Naiserator is a Kubernetes operator that handles the lifecycle of the `CustomResource` called `nais.io/Application`.
The main goal of Naiserator is to simplify application deployment by providing a high-level abstraction tailored for the [NAIS-platform](https://nais.io).
Naiserator supersedes [naisd](https://nais.io).

When an `Application` resource is created in Kubernetes (see
[example application](pkg/apis/naiserator/v1alpha1/application.yaml)),
Naiserator will generate several resources that work together to form a complete deployment:
  * `Deployment` that runs a specified number of application instances,
  * `Service` which points to the application endpoint,
  * `Ingress` adding TLS termination and virtualhost support,
  * `Horizontal pod autoscaler` for automatic application scaling,
  * `Service account` for granting correct permissions to managed resources.
  
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

local development (assumes [Docker Desktop](https://www.docker.com/products/docker-desktop) or [minikube](https://github.com/kubernetes/minikube)
```
kubectl apply -f api/types/v1alpha1/application.yaml
kubectl apply -f examples/nais_example.yaml
go run cmd/naiserator/main.go --logtostderr --kubeconfig=<your kubeconfig file> --bind-address=127.0.0.1:8080
```

### Code generation

In order to use the Kubernetes Go library, we need to use classes that work together with the interfaces in that library.
Those classes are mostly boilerplate code, and to ensure healthy and happy developers, we use code generators for that.

When the CRD changes, or additional Kubernetes resources need to be generated, you have to run code generation:

```
make codegen-crd
make codegen-updater
git add -A
git commit -a -m "Update boilerplate k8s API code"
```

## Differences from previous nais.yaml

* The `redis` field has been removed ([#6][i6])
* The `alerts` field has been removed ([#7][i7])
* Ingresses need to be specified explicitly ([#14][i14])
* The `fasitResources` field has been removed ([#15][i15])

[i6]: https://github.com/nais/naiserator/issues/6
[i7]: https://github.com/nais/naiserator/issues/7
[i14]: https://github.com/nais/naiserator/issues/14
[i15]: https://github.com/nais/naiserator/issues/15

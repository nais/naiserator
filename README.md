# Naiserator

[![CircleCI](https://circleci.com/gh/nais/naiserator/tree/master.svg?style=svg)](https://circleci.com/gh/nais/naiserator/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/nais/naiserator)](https://goreportcard.com/report/github.com/nais/naiserator)

Naiserator is a Kubernetes operator that handles the lifecycle of the custom resource `nais.io/Application`.
The main goal of Naiserator is to simplify application deployment by providing a high-level abstraction tailored for the [NAIS platform](https://nais.io).
Naiserator supersedes [naisd](https://nais.io).

When an `Application` resource is created in Kubernetes,
Naiserator will generate several resources that work together to form a complete deployment:
  * `Deployment` that runs a specified number of application instances
  * `Service` which points to the application endpoint
  * `Horizontal pod autoscaler` for automatic application scaling
  * `Service account` for granting correct permissions to managed resources

Optionally, if enabled in the application manifest, the following resources:
  * `Ingress` adding TLS termination and virtualhost support
  * _Leader election_ sidecar which decides a cluster leader from available pods. This feature also creates a `Role` and `Role binding`.

If Istio support is enabled with `--features.access-policy`:
  * `NetworkPolicy`
  * A set of `VirtualService`
  * `ServiceRole` and `ServiceRoleBinding` resources
  * `ServiceEntry`

These resources will remain in Kubernetes until the `Application` resource is deleted.
Any unneeded resources will be automatically deleted if disabled by feature flags or is lacking in a application manifest.

## Documentation

The entire specification for the manifest is documented in our [doc.nais.io](https://doc.nais.io/in-depth/nais-manifest).

## Deployment

* [Kubernetes](https://kubernetes.io/) v1.11.0 or later

### Installation

You can deploy the most recent release of Naiserator by applying to your cluster:
```
kubectl apply -f hack/resources/
```

## Development

* The [Go](https://golang.org/dl/) programming language, version 1.11 or later
* [goimports](https://godoc.org/golang.org/x/tools/cmd/goimports)
* [Docker Desktop](https://www.docker.com/products/docker-desktop) or other Docker release compatible with Kubernetes
* Kubernetes, either through [minikube](https://github.com/kubernetes/minikube) or a local cluster

[Go modules](https://github.com/golang/go/wiki/Modules)
are used for dependency tracking. Make sure you do `export GO111MODULE=on` before running any Go commands.
It is no longer needed to have the project checked out in your `$GOPATH`.

```
kubectl apply -f config
kubectl apply -f examples/nais.yaml
make local
```

### Kafka & Protobuf

Whenever an Application is synchronized, a [deployment event message](https://github.com/navikt/protos/blob/master/deployment/deployment.proto)
can be sent to a Kafka topic. There's a few prerequisites to develop with this enabled locally:

1. [Protobuf installed](https://github.com/golang/protobuf)
2. An instance of kafka to test against. Use `docker-compose up` to bring up a local instance.
3. Enable this feature by passing `-kafka-enabled=true` when starting Naiserator.

#### Update and compile Protobuf definition
Whenever the Protobuf definition is updated you can update using `make proto`. It will download the definitions, compile and place them in the correct packages.

### Code generation

In order to use the Kubernetes Go library, we need to use classes that work together with the interfaces in that library.
Those classes are mostly boilerplate code, and to ensure healthy and happy developers, we use code generators for that.

When the CRD changes, or additional Kubernetes resources need to be generated, you have to run code generation:

```
make crd
make codegen-crd
make codegen-updater
git add -A
git commit -a -m "Update boilerplate k8s API code"
```

### controller-gen

The tool _controller-gen_ is used by `make crd` to generate a CRD YAML file using the Go type specifications in
`pkg/apis/nais.io/v1alpha1/*_types.go`. This YAML file should not be edited by hand. Any changes needed should
go directly into the Go file as magic annotations.

Check out the [controller-gen documentation](https://book.kubebuilder.io/reference/generating-crd.html) if unsure.

A known working version of controller-gen is `v0.2.1`.

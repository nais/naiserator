# Naiserator

[![Github Actions](https://github.com/nais/naiserator/workflows/Build%20and%20deploy/badge.svg)](https://github.com/nais/naiserator/actions?query=workflow%3A%22Build+and+deploy%22)
[![Go Report Card](https://goreportcard.com/badge/github.com/nais/naiserator)](https://goreportcard.com/report/github.com/nais/naiserator)

tl;dr: _Naiserator is a fancy template engine that runs within Kubernetes_.

Naiserator is a Kubernetes operator that handles the lifecycle of NAIS custom resources, currently `nais.io/Application` and `nais.io/NaisJob`.
The main goal of Naiserator is to simplify application deployment by providing a high-level abstraction tailored for the [NAIS platform](https://nais.io).

When an `Application` resource is created in Kubernetes,
Naiserator will generate several other Kubernetes resources that work together to form a complete deployment.
All of these resources will remain in Kubernetes, until the `Application` resource is deleted, upon which they will be removed.
Additionally, any unneeded resources will be automatically deleted if disabled by feature flags or is lacking in a application manifest.

<!-- For a quick list of generated resources, run:
% rg '^\s*kind' pkg/resourcecreator/testdata | awk '{print $3}' | sort -u
-->

## Generated resources

Kubernetes built-ins:
  * `Deployment`, `Job` or `CronJob` that runs program executables,
  * `HorizontalPodAutoscaler` for automatic application scaling,
  * `Ingress` adding TLS termination and virtualhost support,
  * `NetworkPolicy` for firewall configuration,
  * `PodDisruptionBudget` for controlling how the application should be shut down or restart by Kubernetes,
  * `PodMonitor` for Prometheus integration,
  * `Role` and `RoleBinding` needed for _Leader election_ sidecar,
  * `Secret` for stuff that shouldn't be shared with anyone,
  * `ServiceAccount` for granting correct permissions to managed resources,
  * `Service` which points to the application endpoint.

NAIS resources for external system provisioning:
  * `AivenApplication` for [Aivenator](https://github.com/nais/aivenator),
  * `AzureAdApplication` for [Azurerator](https://github.com/nais/azurerator),
  * `IDPortenClient` and `MaskinportenClient` for [Digdirator](https://github.com/nais/digdirator),
  * `Jwker` for [Jwker](https://github.com/nais/jwker),
  * `Stream` for [Kafkarator](https://github.com/nais/kafkarator).

Google CNRM resources for Google Cloud Platform provisioning:
  * `BigQueryDataset` for BigQuery,
  * `IAMServiceAccount`, `IAMPolicy` and `IAMPolicyMember` for workload identity,
  * `PubSubSubscription` for PubSub,
  * `SQLInstance`, `SQLUser` and `SqlDatabase` for Cloud SQL,
  * `StorageBucket` and `StorageBucketAccessControl` for Storage Buckets.

## Documentation

The entire specification for the manifest is documented in our [doc.nais.io](https://doc.nais.io/nais-application/application/).

## Deployment

Runs on:

* On-premises [Kubernetes](https://kubernetes.io/) v1.21.0 or later
* Google Kubernetes Engine

### Installation

You can deploy the most recent release of Naiserator by applying to your cluster:
```
kubectl apply -f hack/resources/
```

## Development

* The [Go](https://golang.org/dl/) programming language, version 1.18 or later
* [goimports](https://godoc.org/golang.org/x/tools/cmd/goimports)
* [Docker Desktop](https://www.docker.com/products/docker-desktop) or other Docker release compatible with Kubernetes
* Kubernetes, either through [minikube](https://github.com/kubernetes/minikube) or a local cluster

Try these:

```
make test
make golden_file_test
make build
make local
```

### Kafka & Protobuf

Whenever an Application is synchronized, a [deployment event message](https://github.com/navikt/protos/blob/master/deployment/deployment.proto)
can be sent to a Kafka topic. There's a few prerequisites to develop with this enabled locally:

1. [Protobuf installed](https://github.com/golang/protobuf)
2. An instance of kafka to test against. Use `docker-compose up` to bring up a local instance.
3. Enable this feature by passing `-kafka-enabled=true` when starting Naiserator.

#### Update and compile Protobuf definition

Whenever the Protobuf definition is updated you can update using `make proto`. It will download the definitions, compile
and place them in the correct packages. 

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

The CRD spec will be generated in `config/crd/nais.io_applications.yaml`.

Check out the [controller-gen documentation](https://book.kubebuilder.io/reference/generating-crd.html) if unsure.

A known working version of controller-gen is `v0.2.5`. Download with

```
GO111MODULE=off go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.5
```

## Verifying the Naiserator image and its contents

The image is signed "keylessly" (is that a word?) using [Sigstore cosign](https://github.com/sigstore/cosign).
To verify its authenticity run
```
cosign verify \
--certificate-identity "https://github.com/nais/naiserator/.github/workflows/deploy.yaml@refs/heads/master" \
--certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
ghcr.io/nais/naiserator/naiserator@sha256:<shasum>
```

The images are also attested with SBOMs in the [CycloneDX](https://cyclonedx.org/) format.
You can verify these by running
```
cosign verify-attestation --type cyclonedx \
--certificate-identity "https://github.com/nais/naiserator/.github/workflows/deploy.yaml@refs/heads/master" \
--certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
ghcr.io/nais/naiserator/naiserator@sha256:<shasum>
```

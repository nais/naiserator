# Naiserator

[![Github Actions](https://github.com/nais/naiserator/workflows/Build%20and%20deploy/badge.svg)](https://github.com/nais/naiserator/actions?query=workflow%3A%22Build+and+deploy%22)
[![Go Report Card](https://goreportcard.com/badge/github.com/nais/naiserator)](https://goreportcard.com/report/github.com/nais/naiserator)

Naiserator is a Kubernetes operator that handles the lifecycle of `nais.io/Application` and `nais.io/Naisjob`.

The main goal of Naiserator is to simplify application deployment by providing a high-level abstraction tailored for the [Nais platform](https://nais.io).

When an `Application` resource is created in Kubernetes, usually with [Nais deploy](https://github.com/nais/deploy),
Naiserator will generate several other Kubernetes resources that work together to form a complete deployment.
The contents of these resources are heavily dependent on per-cluster and per-application configuration.

Resources will remain in Kubernetes until the `Application` resource is deleted, upon which they will be removed.
Additionally, any unneeded resources will be automatically deleted upon next deploy
if disabled by feature flags or is lacking in a application manifest.

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

Nais resources for external system provisioning:
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

The entire [specification for the manifest](https://doc.nais.io/nais-application/application/) is
generated by Naiserator's companion library, [liberator](https://github.com/nais/liberator),
and committed to the Nais end-user documentation.

## Deployment

Runs on Kubernetes v1.30.0 or later.

When GCP features are enabled, Naiserator must run on Google Kubernetes Engine together with CNRM.

See `charts/naiserator` for a installable Helm chart.

## Development

* The [Go](https://golang.org/dl/) programming language, version indicated by go.mod
* [liberator](https://github.com/nais/liberator)
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

Whenever an Application is deployed, a [deployment event message](https://github.com/navikt/protos/blob/master/deployment/deployment.proto)
is sent to a Kafka topic. There's a few prerequisites to develop with this enabled locally:

1. [Protobuf installed](https://github.com/golang/protobuf)
2. An instance of kafka to test against. Use `docker-compose up` to bring up a local instance.
3. Enable this feature by passing `-kafka-enabled=true` when starting Naiserator.

#### Update and compile Protobuf definition

Whenever the Protobuf definition is updated you can update using `make proto`. It will download the definitions, compile
and place them in the correct packages.

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

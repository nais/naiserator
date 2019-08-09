# Naiserator

[![CircleCI](https://circleci.com/gh/nais/naiserator/tree/master.svg?style=svg)](https://circleci.com/gh/nais/naiserator/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/nais/naiserator)](https://goreportcard.com/report/github.com/nais/naiserator)

Naiserator is a Kubernetes operator that handles the lifecycle of the custom resource `nais.io/Application`.
The main goal of Naiserator is to simplify application deployment by providing a high-level abstraction tailored for the [NAIS platform](https://nais.io).
Naiserator supersedes [naisd](https://nais.io).

When an `Application` resource is created in Kubernetes (see
[example application](examples/nais.yaml)),
Naiserator will generate several resources that work together to form a complete deployment:
  * `Deployment` that runs a specified number of application instances,
  * `Service` which points to the application endpoint,
  * `Ingress` adding TLS termination and virtualhost support,
  * `Horizontal pod autoscaler` for automatic application scaling,
  * `Service account` for granting correct permissions to managed resources.

These resources will remain in Kubernetes until the `Application` resource is deleted.

## `nais.io/Application` spec

| Parameter | Description | Default | Required |
| --------- | ----------- | ------- | :--------: |
| metadata.name | Name of the application | | x |
| metadata.namespace | Which namespace the application will be deployed to | | x |
| metadata.labels.team | [mailnick/tag](https://github.com/nais/doc/blob/master/content/getting-started/teamadministration.md) | | x |
| spec.image | Docker image location, including version | | x |
| spec.port | The HTTP port exposed by the container | 8080 | |
| spec.strategy.type | Specifies the strategy used to replace old Pods by new ones | RollingUpdate |
| spec.liveness.path | Path of the [liveness probe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/) | | x |
| spec.liveness.port | Port for liveness probe | `.spec.port` |
| spec.liveness.initialDelay | Number of seconds after the container has started before liveness probes are initiated | 20 |
| spec.liveness.timeout | Number of seconds after which the probe times out | 1 |
| spec.liveness.periodSeconds | How often (in seconds) to perform the probe | 10 |
| spec.liveness.failureThreshold | When a Pod starts and the probe fails, Kubernetes will try `failureThreshold` times before giving up. Giving up in case of liveness probe means restarting the Pod. In case of readiness probe the Pod will be marked Unready | 3 |
| spec.readiness.path | Path of the [readiness probe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/) | | x |
| spec.readiness.port | Port for readiness probe | `.spec.port` |
| spec.readiness.initialDelay | Number of seconds after the container has started before readiness probes are initiated | 20 |
| spec.readiness.timeout | Number of seconds after which the probe times out | 1 |
| spec.replicas.min | Minimum number of replicas | 2 |
| spec.replicas.max | Maximum number of replicas | 4 |
| spec.cpuThresholdPercentage | Total CPU percentage threshold on deployment, at which point it will increase number of pods if `current < max`. See [container lifecycle hooks documentation](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/) |
| spec.prometheus.enabled | If true, the pod will be scraped for metrics by Prometheus | false |
| spec.prometheus.port | Path to Prometheus metrics | `.spec.port` |
| spec.prometheus.path | Path to Prometheus metrics | /metrics |
| spec.resources | See [compute resources guide](http://kubernetes.io/docs/user-guide/compute-resources/) | |
| spec.resources.limits.cpu | App will have its CPU usage throttled if exceeding this limit | 500m |
| spec.resources.limits.memory | App will be killed if exceeding this limit | 512Mi |
| spec.resources.requests | App is guaranteed the requested resources and will be scheduled on nodes with at least this amount of resources available | |
| spec.resources.requests.cpu | Guaranteed amount of CPU | 200m |
| spec.resources.requests.memory | Guaranteed amount of memory | 256Mi |
| spec.ingresses | List of ingress URLs that will route HTTP traffic to the application | |
| spec.secrets[].name | (GCP only, not available on-prem) Name of secret (must exist in namespace) | | x |
| spec.secrets[].type | (GCP only, not available on-prem) How the secrets is exposed to the pod. Valid options is `env` and `files`. Selecting `env` will expose all variables in secret as environment variables and `files` will expose the secrets as files under `spec.secrets[].mountPath` | `env` | |
| spec.secrets[].mountPath | (GCP only, not available on-prem) Path to where secret files will be mounted (only valid for secret type `files`) | /var/run/secrets | |
| spec.vault.enabled | If set to true, fetch secrets from [Vault](https://github.com/nais/doc/tree/master/content/secrets) and inject into the pods | false | |
| spec.vault.sidecar | If set to true, will extend tokens time to live | false | |
| spec.vault.paths | Overriding the `paths` array is optional, and will give you fine-grained control over which vault paths that will be mounted on the file system. | | |
| spec.vault.paths.[].kvPath | Path to Vault key/value store that should be mounted into the file system | /kv/environment/zone/application/namespace | |
| spec.vault.paths.[].mountPath | File system path that the secrets will be mounted into | /var/run/secrets/nais.io/vault | |
| spec.configMaps.files | List of ConfigMap resources that will have their contents mounted into the container as files. Files appear as `/var/run/configmaps/<name>/<key>`. | |
| spec.env[].name | Environment variable name | | x |
| spec.env[].value | Environment variable value | | x |
| spec.preStopHookPath | A HTTP GET will be issued to this endpoint at least once before the pod is terminated | /stop | |
| spec.leaderElection | If true, a HTTP endpoint will be available at `$ELECTOR_PATH` that returns the current leader | false | |
| spec.webproxy | Expose web proxy configuration to the application using the `$HTTP_PROXY`, `$HTTPS_PROXY` and `$NO_PROXY` environment variables | false | |
| spec.logformat | Format of the logs from the container, if not in plain text or JSON | accesslog | |
| spec.logtransform | The transformation of the logs, if they should be handled differently than plain text or JSON | dns_loglevel | |
| spec.secureLogs.enabled | If true, mount a volume for secure logs in the pod | false | |
| spec.service.port | Port for the default service | 80 |
| spec.skipCaBundle | If true, no certificate authority bundle will be injected | false | |
| spec.accessPolicy| Default will not allow any traffic to or from application. Access policy is currently supported in GKE clusters, only. Read more in our [documentation](https://github.com/nais/doc/tree/master/content/drafts/access-policies.md) |  | |
| spec.accessPolicy.inbound.rules | List of services to allow traffic from |  |
| spec.accessPolicy.inbound.rules[].application | Name of the application to allow traffic from |  | x |
| spec.accessPolicy.inbound.rules.namespace | Namespace to application to allow traffic from | metadata.namespace |
| spec.accessPolicy.outbound.rules | List of services to allow traffic to |  |
| spec.accessPolicy.outbound.rules[].application | Name of the other service to allow traffic to |  | x |
| spec.accessPolicy.outbound.rules[].namespace | Namespace of the other service to allow traffic to |  | metadata.namespace |
| spec.accessPolicy.outbound.external | List of services outside cluster to allow traffic to |  |
| spec.accessPolicy.outbound.external[].host | URL to service outside cluster |  |  x |

In the [examples directory](./examples) you can see a [typical `nais.yaml` file](./examples/nais.yaml)

## Default environment variables

These environment variables will be injected into your application container

| variable          | example         | source                            |
| ----------------- | --------------- | --------------------------------- |
| NAIS_APP_NAME     | myapp           | metadata.name from nais.yaml      |
| NAIS_NAMESPACE    | default         | metadata.namespace from nais.yaml |
| NAIS_APP_IMAGE    | navikt/myapp:69 | spec.image from nais.yaml         |
| NAIS_CLUSTER_NAME | prod-fss        | naiserator runtime context        |

## Migrating from naisd

In order to switch from naisd to Naiserator, you need to complete a few migration tasks.
See [migration from naisd to naiserator](https://github.com/nais/doc/blob/master/content/deploy/migrating_from_naisd.md) for a detailed explanation
of the steps involved.

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

A known working version of controller-gen is `v0.2.0-beta.3`.

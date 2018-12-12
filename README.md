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

## `nais.io/Application` spec

| Parameter | Description | Default | Required |
| --------- | ----------- | ------- | :--------: |
| metadata.name | Name of the application | | x |
| metadata.namespace | Which namespace the application will be deployed to | | x |
| metadata.labels.team | [mailnick/tag](https://github.com/nais/doc/blob/master/content/getting-started/teamadministration.md) | | x |
| spec.image | Docker image location | | x |
| spec.port | The port exposed by the container | | x |
| spec.liveness.path | Path of the [liveness probe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/) | | x |
| spec.liveness.initialDelay | Number of seconds after the container has started before liveness probes are initiated | 20 |
| spec.liveness.timeout | Number of seconds after which the probe times out. | 1 |
| spec.liveness.periodSeconds | How often (in seconds) to perform the probe | 10 |
| spec.liveness.failureThreshold | When a Pod starts and the probe fails, Kubernetes will try `failureThreshold` times before giving up. Giving up in case of liveness probe means restarting the Pod. In case of readiness probe the Pod will be marked Unready | 3 |
| spec.readiness.path | Path of the [readiness probe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/) | | x |
| spec.readiness.initialDelay | Number of seconds after the container has started before readiness probes are initiated | 20 |
| spec.readiness.timeout | Number of seconds after which the probe times out. | 1 |
| spec.replicas.min | Minimum number of replicas | 2 |
| spec.replicas.max | Maximum number of replicas | 4 |
| spec.cpuThresholdPercentage | Total cpu percentage threshold on deployment, at which point it will increase number of pods if current < max. See [doc](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/) for details
| spec.prometheus | | |  |
| spec.prometheus.enabled | If true, the pod will be scraped for metrics by Prometheus | false |
| spec.prometheus.path | Path to Prometheus metrics | /metrics |
| spec.resources| see [guide](http://kubernetes.io/docs/user-guide/compute-resources/) | |  |
| spec.resources.limits.cpu | App will have its cpu usage throttled if exceeding this limit | 500m |
| spec.resources.limits.memory | App will be killed if exceeding this limit | 512Mi |
| spec.resources.requests | App is guaranteed the requested resources and will be shceduled on nodes with atleast this amount of resources available | |
| spec.resources.requests.cpu | Guaranteed amount of CPU | 200m |
| spec.resources.requests.memory | Guaranteed amount of memory | 256Mi |
| spec.ingresses | List of ingress URLs that will route HTTP traffic to the application | |  |
| spec.secrets | If set to true, fetch secrets from [Vault](https://github.com/nais/doc/tree/master/content/secrets) and inject into the pods | false |  | 
| spec.configMaps.files | List of configMaps that will have their data mounted into the container as files. | |  |
| spec.env | List of name and value that will become environment variables in the container |  |  |
| spec.preStopHookPath | A HTTP GET will be issued to this endpoint at least once before the pod is terminated | /stop |  |
| spec.leaderElection | If true, a http endpoint will be available at `$ELECTOR_PATH` that returns the current leader | False |  |
| spec.webproxy | Expose webproxy configuration to the application using the HTTP_PROXY, HTTPS_PROXY and NO_PROXY environment variables | false |  |
| spec.logformat | The format of the logs from the container if the logs should be handled differently than plain text or json | accesslog |  |
| spec.logtransfrom | The tranormation of the logs, if they should be handled differently than plain text or json | dns_loglevel |  |

In the [examples directory](./examples) you can see a [typical `nais.yaml` file](./examples/nais.yaml)


## Migrating from naisd

In order to switch from naisd to Naiserator, you need to complete a few migration tasks.
See [migration from naisd to naiserator](https://github.com/nais/doc/blob/master/content/deploy/migrating_from_naisd.md) for a detailed explanation
of the steps involved.
  
## Prerequisites

### Deployment

* [Kubernetes](https://kubernetes.io/) v1.11.0 or later

### Development

* The [Go](https://golang.org/dl/) programming language, version 1.11 or later
* [Docker Desktop](https://www.docker.com/products/docker-desktop) or other Docker release compatible with Kubernetes
* Kubernetes, either through [minikube](https://github.com/kubernetes/minikube) or a local cluster

## Installation

You can deploy the most recent release of Naiserator by applying to your cluster:
```
kubectl apply -f hack/resources/
```

## Development

[Go modules](https://github.com/golang/go/wiki/Modules)
are used for dependency tracking. Make sure you do `export GO111MODULE=on` before running any Go commands.
It is no longer needed to have the project checked out in your `$GOPATH`.

```
kubectl apply -f pkg/apis/naiserator/v1alpha1/application.yaml
kubectl apply -f examples/app.yaml
make local
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

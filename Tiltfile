load('ext://cert_manager', 'deploy_cert_manager')
load('ext://helm_resource', 'helm_resource', 'helm_repo')
load('ext://local_output', 'local_output')
load('ext://namespace', 'namespace_create')

APP_NAME = "naiserator"


def ignore_rules():
    return str(read_file(".dockerignore")).split("\n")


deploy_cert_manager()

helm_repo('aiven', 'https://aiven.github.io/aiven-charts')
helm_resource('aiven-operator-crds', 'aiven/aiven-operator-crds', resource_deps=['aiven'], pod_readiness="ignore")

helm_repo('prometheus', 'https://prometheus-community.github.io/helm-charts')
helm_resource('prometheus-operator-crds', 'prometheus/prometheus-operator-crds', resource_deps=['prometheus'],
              pod_readiness="ignore")

namespace_create("nais-system")
namespace_create("basseng", labels=["team: basseng", "nais.io/type: workload"])
namespace_create("pg-basseng", labels=["team: basseng", "nais.io/type: postgres"])

# Load liberator charts, assuming liberator checked out next to naiserator
# Automatically generate updated CRDs from the liberator code when the apis change, and apply them to the cluster
local_resource(
    "liberator-chart",
    cmd="make generate",
    dir="../liberator",
    ignore=["../liberator/**/zz_generated.deepcopy.go"],
    deps=["../liberator/pkg/apis"],
)
k8s_yaml(helm("../liberator/charts", name="nais-crds"))
liberator_objects = [
    "aivenapplications.aiven.nais.io:CustomResourceDefinition:default",
    "bigquerydatasets.google.nais.io:CustomResourceDefinition:default",
    "streams.kafka.nais.io:CustomResourceDefinition:default",
    "topics.kafka.nais.io:CustomResourceDefinition:default",
    "applications.nais.io:CustomResourceDefinition:default",
    "azureadapplications.nais.io:CustomResourceDefinition:default",
    "idportenclients.nais.io:CustomResourceDefinition:default",
    "images.nais.io:CustomResourceDefinition:default",
    "jwkers.nais.io:CustomResourceDefinition:default",
    "maskinportenclients.nais.io:CustomResourceDefinition:default",
    "naisjobs.nais.io:CustomResourceDefinition:default",
]
k8s_resource(
    new_name="nais-crds",
    objects=liberator_objects,
    resource_deps=["liberator-chart"],
)

# Load charts, assuming helm-charts checked out next to naiserator
k8s_yaml(helm("../helm-charts/features/postgres-operator", name="postgres-operator", namespace="nais-system", set="fasit.env.kind=management,apiServerCIDR=10.96.0.1/32"))
k8s_resource(
    workload="postgres-operator",
    resource_deps=["nais-crds"],
)


# Create a tempdir for naiserator configs
naiserator_dir = "/tmp/tilt-naiserator"
local("mkdir -p {}".format(naiserator_dir))

# Copy tilt spesific naiserator config to tempdir for naiserator to use
local_resource(
    "naiserator-config",
    cmd="cp ./hack/tilt-naiserator-config.yaml {}/naiserator.yaml".format(naiserator_dir),
    deps=["hack/tilt-naiserator-config.yaml"],
)

# Ensure we save the current kube context to a file for naiserator
# This is so we don't accidentally switch context if other tools change the current context after startup
# Falls apart if the Tiltfile is updated, as that copies the kubeconfig again.
# See https://github.com/tilt-dev/tilt/issues/6295
local_resource(
    "naiserator-kubeconfig",
    cmd="kubectl config view --minify --flatten > {}/kubeconfig".format(naiserator_dir),
)

# Start naiserator locally, so changes are detected and rebuilt automatically
local_resource(
    "naiserator",
    cmd="go build -o cmd/naiserator/naiserator ./cmd/naiserator",
    serve_cmd="{}/cmd/naiserator/naiserator --kubeconfig={}/kubeconfig".format(config.main_dir, naiserator_dir),
    deps=["cmd/naiserator/naiserator.go", "go.mod", "go.sum", "pkg", "/tmp/naiserator.yaml"],
    resource_deps=[
        "nais-crds",
        "aiven-operator-crds",
        "prometheus-operator-crds",
        "naiserator-config",
        "naiserator-kubeconfig",
        "postgres-operator",
    ],
    ignore=ignore_rules(),
    serve_dir=naiserator_dir,
)


config.set_enabled_resources([
    "aiven",
    "aiven-operator-crds",
    "prometheus",
    "prometheus-operator-crds",
    "liberator-chart",
    "nais-crds",
    "naiserator-config",
    "naiserator-kubeconfig",
    "postgres-operator",
])

load('ext://cert_manager', 'deploy_cert_manager')
load('ext://helm_resource', 'helm_resource', 'helm_repo')
load('ext://local_output', 'local_output')
load('ext://restart_process', 'docker_build_with_restart')

APP_NAME="naiserator"


def api_server_ip():
    cmd = ["kubectl", "get", "service", "--context", k8s_context(), "--namespace", "default", "kubernetes", "-o", "jsonpath='{.spec.clusterIP}'"]
    return str(local(cmd, quiet=True))[1:-1]


def ignore_rules():
    return str(read_file(".dockerignore")).split("\n")

deploy_cert_manager()

helm_repo('aiven', 'https://aiven.github.io/aiven-charts')
helm_resource('aiven-operator-crds', 'aiven/aiven-operator-crds', resource_deps=['aiven'], pod_readiness="ignore")

helm_repo('prometheus', 'https://prometheus-community.github.io/helm-charts')
helm_resource('prometheus-operator-crds', 'prometheus/prometheus-operator-crds', resource_deps=['prometheus'], pod_readiness="ignore")

# Load liberator charts, assuming liberator checked out next to naiserator
local_resource("liberator-chart",
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

local_resource("naiserator-config",
    cmd="cp ./hack/tilt-naiserator-config.yaml /tmp/naiserator.yaml",
    deps=["hack/tilt-naiserator-config.yaml"],
)

local_resource("naiserator",
    cmd="go build -o cmd/naiserator/naiserator ./cmd/naiserator",
    serve_cmd="{}/cmd/naiserator/naiserator".format(config. main_dir),
    deps=["cmd/naiserator/naiserator.go", "go.mod", "go.sum", "pkg", "/tmp/naiserator.yaml"],
    resource_deps=["nais-crds", "aiven-operator-crds", "prometheus-operator-crds", "naiserator-config"],
    ignore=ignore_rules(),
    serve_dir="/tmp",
)

NAME        := naiserator
TAG         := navikt/${NAME}
LATEST      := ${TAG}:latest
ROOT_DIR    := $(shell git rev-parse --show-toplevel)

# This is used for Docker
arch        := $(shell uname -m | sed s/aarch64/arm64/ | sed s/x86_64/amd64/)
os          := $(shell uname -s | tr '[:upper:]' '[:lower:]')

# This works locally, but not in CI
ENVTEST_VERSION ?= release-0.23
ENVTEST_K8S_VERSION ?= 1.30.0
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

ENVTEST ?= $(LOCALBIN)/setup-envtest

.PHONY: build docker docker-push local install test kubebuilder

build:
	go build -o cmd/naiserator/naiserator ./cmd/naiserator
	go build -o cmd/naiserator_webhook/naiserator_webhook ./cmd/naiserator_webhook

docker:
	docker image build -t ${TAG}:$(shell ./version.sh) -t ${TAG} -t ${NAME} -t ${LATEST} -f Dockerfile .

docker-push:
	docker image push ${TAG}:$(shell /bin/cat ./version)
	docker image push ${LATEST}

local:
	go run cmd/naiserator/main.go --kubeconfig=${KUBECONFIG} --bind=127.0.0.1:8080

install:
	cd cmd/naiserator && go install

test: setup-envtest
	go test ./... -count=1 --coverprofile=cover.out

golden_file_test:
	go test ./pkg/resourcecreator/resourcecreator_golden_files_test.go -count=1

##@ Dependencies
setup-envtest: envtest ## Download the binaries required for ENVTEST in the local bin directory.
	@echo "Setting up envtest binaries for Kubernetes version $(ENVTEST_K8S_VERSION)..."
	@$(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path || { \
		echo "Error: Failed to set up envtest binaries for version $(ENVTEST_K8S_VERSION)."; \
		exit 1; \
	}

envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

VER ?= test-image
test-image:
	docker image build --platform=linux/amd64 -t europe-north1-docker.pkg.dev/nais-io/nais/images/naiserator:$(VER) .
	docker image push europe-north1-docker.pkg.dev/nais-io/nais/images/naiserator:$(VER)

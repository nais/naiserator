NAME        := naiserator
TAG         := europe-north1-docker.pkg.dev/nais-io/nais/feature/${NAME}
LATEST      := ${TAG}:work-in-progress
ROOT_DIR    := $(shell git rev-parse --show-toplevel)
K8S_VERSION := 1.30.0
arch        := $(shell uname -m | sed s/aarch64/arm64/ | sed s/x86_64/amd64/)
os          := $(shell uname -s | tr '[:upper:]' '[:lower:]')
testbin_dir := ./.testbin/
tools_archive := kubebuilder-tools-${K8S_VERSION}-$(os)-$(arch).tar.gz

PROTOC = $(shell which protoc)

.PHONY: build docker docker-push local install test proto kubebuilder

build:
	go build -o cmd/naiserator/naiserator ./cmd/naiserator
	go build -o cmd/naiserator_webhook/naiserator_webhook ./cmd/naiserator_webhook

docker:
	docker buildx build --platform linux/amd64 -t ${TAG}:$(shell ./version.sh) -t ${TAG} -t ${NAME} -t ${LATEST} -f Dockerfile .

docker-push:
	docker image push ${TAG}:$(shell /bin/cat ./version)
	docker image push ${LATEST}

local:
	go run cmd/naiserator/main.go --kubeconfig=${KUBECONFIG} --bind=127.0.0.1:8080

install:
	cd cmd/naiserator && go install

test: kubebuilder
	go test ./... -count=1 --coverprofile=cover.out

golden_file_test:
	go test ./pkg/resourcecreator/resourcecreator_golden_files_test.go -count=1

kubebuilder: $(testbin_dir)/$(tools_archive)
	tar -xzf $(testbin_dir)/$(tools_archive) --strip-components=2 -C $(testbin_dir)
	chmod -R +x $(testbin_dir)

$(testbin_dir)/$(tools_archive):
	mkdir -p $(testbin_dir)
	wget -q --directory-prefix=$(testbin_dir) "https://storage.googleapis.com/kubebuilder-tools/$(tools_archive)"

proto:
	wget -O pkg/event/event.proto https://raw.githubusercontent.com/navikt/protos/master/deployment/event.proto
	$(PROTOC) --go_opt=Mpkg/event/event.proto=github.com/nais/naiserator/pkg/deployment,paths=source_relative --go_out=. pkg/event/event.proto
	rm -f pkg/event/event.proto

install-protobuf-go:
	go install google.golang.org/protobuf/cmd/protoc-gen-go
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc

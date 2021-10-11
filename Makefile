NAME        := naiserator
TAG         := navikt/${NAME}
LATEST      := ${TAG}:latest
ROOT_DIR    := $(shell git rev-parse --show-toplevel)
K8S_VERSION := 1.19.0
arch        := amd64
os          := $(shell uname -s | tr '[:upper:]' '[:lower:]')

PROTOC = $(shell which protoc)

.PHONY: build docker docker-push local install test proto kubebuilder

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

test:
	go test ./... -count=1 --coverprofile=cover.out

golden_file_test:
	go test ./pkg/resourcecreator/resourcecreator_golden_files_test.go -count=1

kubebuilder:
	test -d /usr/local/kubebuilder || (sudo mkdir -p /usr/local/kubebuilder && sudo chown "${USER}" /usr/local/kubebuilder)
	curl -L "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-${K8S_VERSION}-$(os)-$(arch).tar.gz" | tar -xz -C /usr/local
	curl -L -o /usr/local/kubebuilder/bin/kubebuilder https://github.com/kubernetes-sigs/kubebuilder/releases/download/v3.1.0/kubebuilder_$(os)_$(arch)
	chmod +x /usr/local/kubebuilder/bin/*

proto:
	wget -O pkg/event/event.proto https://raw.githubusercontent.com/navikt/protos/master/deployment/event.proto
	$(PROTOC) --go_opt=Mpkg/event/event.proto=github.com/nais/naiserator/pkg/deployment,paths=source_relative --go_out=. pkg/event/event.proto
	rm -f pkg/event/event.proto

install-protobuf-go:
	go install google.golang.org/protobuf/cmd/protoc-gen-go
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc

NAME       := naiserator
TAG        := navikt/${NAME}
LATEST     := ${TAG}:latest
ROOT_DIR   := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
arch       := amd64
os         := $(shell uname -s | tr '[:upper:]' '[:lower:]')

PROTOC = $(shell which protoc)
PROTOC_GEN_GO = $(shell which protoc-gen-go)

.PHONY: build docker docker-push local install test crd codegen-crd codegen-updater proto

build:
	cd cmd/naiserator && go build

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
	curl -L https://go.kubebuilder.io/dl/2.3.1/${os}/${arch} | tar -xz -C /tmp/
	mv /tmp/kubebuilder_2.3.1_${os}_${arch} /usr/local/kubebuilder

proto:
	wget https://raw.githubusercontent.com/navikt/protos/master/deployment/event.proto
	$(PROTOC) --plugin=$(PROTOC_GEN_GO) --go_out=. event.proto
	mv event.pb.go pkg/event/
	rm -f event.proto

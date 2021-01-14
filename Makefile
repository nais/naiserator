NAME       := naiserator
TAG        := navikt/${NAME}
LATEST     := ${TAG}:latest
ROOT_DIR   := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

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

crd:
	controller-gen "crd:trivialVersions=true" paths=github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1 output:crd:artifacts:config=config/crd

codegen-crd:
	${ROOT_DIR}/hack/update-codegen.sh

codegen-updater:
	go generate ${ROOT_DIR}/hack/generator/updater.go | goimports > ${ROOT_DIR}/updater/zz_generated.go

proto:
	wget https://raw.githubusercontent.com/navikt/protos/master/deployment/event.proto
	$(PROTOC) --plugin=$(PROTOC_GEN_GO) --go_out=. event.proto
	mv event.pb.go pkg/event/
	rm -f event.proto

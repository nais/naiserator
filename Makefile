NAME       := naiserator
TAG        := navikt/${NAME}
LATEST     := ${TAG}:latest
KUBECONFIG := ${HOME}/.kube/config
GO_IMG     := golang:1.11
GO         := docker run --rm -v ${PWD}:/go/src/github.com/nais/naiserator -w /go/src/github.com/nais/naiserator ${GO_IMG} go
ROOT_DIR   := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))


.PHONY: build docker local install docker docker-push linux test

build:
	cd cmd/naiserator && go build

docker:
	docker image build -t ${TAG}:$(shell /bin/cat ./version) -t ${TAG} -t ${NAME} -t ${LATEST} -f Dockerfile .

docker-push:
	docker image push ${TAG}:$(shell /bin/cat ./version)
	docker image push ${LATEST}

local:
	go run cmd/naiserator/main.go --logtostderr --kubeconfig=${KUBECONFIG} --bind-address=127.0.0.1:8080

install:
	export GO111MODULE=on && go mod vendor	

test:
	${GO} test ./... --coverprofile=cover.out

codegen:
	${ROOT_DIR}/hack/update-codegen.sh

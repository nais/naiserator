.PHONY: all build docker

all: build

build:
	cd cmd/naiserator && go build

docker:
	docker build -t nais/naiserator .

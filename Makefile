.PHONY: all build docker

all: build

build:
	go build

docker:
	docker build -t nais/naiserator .

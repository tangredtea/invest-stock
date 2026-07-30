.PHONY: preflight test vet race build docker-build release-check run-web

VERSION ?= $(shell git describe --tags --always --dirty)
REVISION ?= $(shell git rev-parse HEAD)
CREATED ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

test:
	go test ./...

vet:
	go vet ./...

race:
	go test -race ./...

preflight: test vet race

build:
	go build -trimpath -o bin/invest-stock ./cmd/web

docker-build:
	docker build --build-arg VERSION="$(VERSION)" --build-arg REVISION="$(REVISION)" --build-arg CREATED="$(CREATED)" --tag "invest-stock:$(VERSION)" .

release-check:
	@printf '%s\n' "$(VERSION)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$$'
	$(MAKE) preflight
	$(MAKE) build
	$(MAKE) docker-build

run-web:
	go run ./cmd/web

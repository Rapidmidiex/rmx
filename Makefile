SHELL=/bin/bash -e -o pipefail
PWD = $(shell pwd)
GO_BUILD= go build
GOFLAGS= CGO_ENABLED=0

## help: Print this help message
.PHONY: help
help:
	@echo "Usage:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' |  sed -e 's/^/ /'

## test: Run tests
.PHONY: test
test:
	go test -race -v ./...

## cover: Run tests and show coverage result
.PHONY: cover
cover:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

## tidy: Cleanup and download missing dependencies
.PHONY: tidy
tidy:
	go mod tidy
	go mod verify

## vet: Examine Go source code and reports suspicious constructs
.PHONY: vet
vet:
	go vet ./...

## fmt: Format all go source files
.PHONY: fmt
fmt:
	go fmt ./...

## build_server: Build server binary into bin/ directory
.PHONY: build_server
build_server:
	$(GOFLAGS) $(GO_BUILD) -a -v -ldflags="-w -s" -o bin/server cmd/server/main.go

## build_cli: Build cli binary into bin/ directory
.PHONY: build_cli
build_cli:
	$(GOFLAGS) $(GO_BUILD) -a -v -ldflags="-w -s" -o bin/cli cmd/cli/main.go

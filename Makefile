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
	make secrets
	mkdir -p ./bin
	$(GOFLAGS) $(GO_BUILD) -a -v -ldflags="-w -s" -o bin/rmx-server cmd/server/main.go

.PHONY: secrets
secrets:
	sh scripts/secrets.sh

# --host should be from ENV
.PHONY: tls
tls:
	go run /usr/local/g/src/crypto/tls/generate_cert.go --host=$(HOSTNAME)

.PHONY: postgres
postgres:
	sh scripts/postgres.sh
	
.PHONY: createdb
createdb:
	sh scripts/createdb.sh

.PHONY: dropdb
dropdb:
	sh scripts/dropdb.sh

.PHONY: migrateup
migrateup:
	sh scripts/migrateup.sh

.PHONY: migratedown
migratedown:
	sh scripts/migratedown.sh

.PHONY: nats
nats:
	sh scripts/nats.sh

.PHONY: sqlc
sqlc:
	sqlc generate

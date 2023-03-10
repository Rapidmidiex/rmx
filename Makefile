SHELL=/bin/bash -e -o pipefail
PWD = $(shell pwd)
GO_BUILD= go build
GOFLAGS= CGO_ENABLED=0

PG_DB=rmx-dev-test
PG_USER=rmx
PG_PASSWORD=postgrespw
PG_HOST=localhost
PG_PORT=5432

PG_CONN_STRING="postgresql://$(PG_USER):$(PG_PASSWORD)@$(PG_HOST):$(PG_PORT)/$(PG_DB)?sslmode=disable"

PG_CONTAINER_NAME=postgres-rmx


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
.PHONY: build
build:
	$(GOFLAGS) $(GO_BUILD) -a -v -ldflags="-w -s" -o bin/rmx-server cmd/*.go

# --host should be from ENV
.PHONT: tls
tls:
	go run /usr/local/go/src/crypto/tls/generate_cert.go --host=$(HOSTNAME)

.PHONY: postgres
postgres:
	docker rm -f postgres-rmx &> /dev/null || true
	docker run --name $(PG_CONTAINER_NAME) -e POSTGRES_PASSWORD=$(PG_PASSWORD) -p $(PG_PORT):5432 -e POSTGRES_USER=$(PG_USER) -d postgres:14.6-alpine

.PHONY: createdb
createdb:
	docker exec -it $(PG_CONTAINER_NAME) createdb --username=$(PG_USER) --owner=$(PG_USER) $(PG_DB)

.PHONY: dropdb
dropdb:
	docker exec -it $(PG_CONTAINER_NAME) dropdb --username=$(PG_USER) $(PG_DB)

.PHONY: migrateup
migrateup:
	migrate -path internal/jam/postgres/migration -database $(PG_CONN_STRING) -verbose up

.PHONY: migratedown
migratedown:
	migrate -path internal/jam/postgres/migration -database $(PG_CONN_STRING) -verbose down

.PHONY: sqlc
sqlc:
	sqlc generate

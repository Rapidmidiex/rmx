SHELL=/bin/bash -e -o pipefail
PWD = $(shell pwd)
GO_BUILD= go build
GOFLAGS= CGO_ENABLED=0

POSTGRES_DB = rmx-dev
POSTGRES_USER = rmx
POSTGRES_PASSWORD = postgrespw

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
	docker run --name postgres-rmx -e POSTGRES_PASSWORD=$(POSTGRES_PASSWORD) -p 5432:5432 -e POSTGRES_USER=$(POSTGRES_USER) -d postgres:14.6-alpine

.PHONY: createdb
createdb:
	docker exec -it  postgres-rmx createdb --username=$(POSTGRES_USER) --owner=$(POSTGRES_USER) $(POSTGRES_DB)

.PHONY: dropdb
dropdb:
	docker exec -it  postgres-rmx dropdb --username=$(POSTGRES_USER) $(POSTGRES_DB)

.PHONY: migrateup
migrateup:
	migrate -path internal/db/migration -database "postgresql://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:5432/$(POSTGRES_DB)?sslmode=disable" -verbose up

.PHONY: migratedown
migratedown:
	migrate -path store/migration -database "postgresql://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:5432/$(POSTGRES_DB)?sslmode=disable" -verbose down

.PHONY: sqlc
sqlc:
	sqlc generate

#!/bin/bash
. ./scripts/env.sh
. ./scripts/select_container.sh
if ! [ -x "$(command -v migrate)" ]; then
  echo 'Warning: go-migrate is not installed. installing...' >&2
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
fi
migrate -path internal/db/migration -database $PG_CONN_STRING -verbose up
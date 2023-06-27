#!/bin/bash
. ./scripts/select_container.sh
. ./scripts/env.sh
if ! [ -x "$(command -v git)" ]; then
  echo 'Warning: go-migrate is not installed. installing...' >&2
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
fi
migrate -path internal/db/migration -database $PG_CONN_STRING -verbose down

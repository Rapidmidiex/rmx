#!/bin/bash
. ./scripts/env.sh
. ./scripts/select_container.sh
$CONTAINER rm -f $PG_CONTAINER_NAME &> /dev/null || true
$CONTAINER run --name $PG_CONTAINER_NAME -e POSTGRES_PASSWORD=$PG_PASSWORD -p $PG_PORT:5432 -e POSTGRES_USER=$PG_USER -d postgres:14.6-alpine
while !</dev/tcp/$PG_HOST/$PG_PORT; do sleep 1; done; $CONTAINER exec $PG_CONTAINER_NAME createdb --username=$PG_USER --owner=$PG_USER $PG_DB
if ! [ -x "$(command -v migrate)" ]; then
  echo 'Warning: go-migrate is not installed. installing...' >&2
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
fi
migrate -path internal/db/migration -database $PG_CONN_STRING -verbose up
$CONTAINER rm -f $NATS_CONTAINER_NAME &> /dev/null || true
$CONTAINER run --name $NATS_CONTAINER_NAME -p $NATS_PORT:4222 -d nats:latest -js 

echo "setup finished."

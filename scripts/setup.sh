#!/bin/bash
CONTAINER="docker"
PG_DB="rmx-dev-test"
PG_USER="rmx"
PG_PASSWORD="postgrespw"
PG_HOST="localhost"
PG_PORT=5432
PG_CONN_STRING="postgresql://$PG_USER:$PG_PASSWORD@$PG_HOST:$PG_PORT/$PG_DB?sslmode=disable"
PG_CONTAINER_NAME="postgres-rmx"
NATS_CONTAINER_NAME="nats-rmx"
NATS_PORT=4222
# choose container
read -p "Choose 'docker' or 'podman' to use (default: docker): " CONTAINER
case $CONTAINER in
    docker) echo "Using docker" ;;
    podman) echo "Using podman" ;;
    *) echo "Unrecognized selection: $CONTAINER"
        exit 
        ;;
esac

$CONTAINER rm -f $PG_CONTAINER_NAME &> /dev/null || true
$CONTAINER run --name $PG_CONTAINER_NAME -e POSTGRES_PASSWORD=$PG_PASSWORD -p $PG_PORT:5432 -e POSTGRES_USER=$PG_USER -d postgres:14.6-alpine

$CONTAINER start $PG_CONTAINER_NAME 
$CONTAINER exec -it $PG_CONTAINER_NAME createdb --username=$PG_USER --owner=$PG_USER $PG_DB

if ! [ -x "$(command -v git)" ]; then
  echo 'Warning: go-migrate is not installed. installing...' >&2
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
fi
migrate -path internal/db/migration -database $PG_CONN_STRING -verbose up

$CONTAINER rm -f $NATS_CONTAINER_NAME &> /dev/null || true
$CONTAINER run --name $NATS_CONTAINER_NAME -p $NATS_PORT:4222 -d nats:latest -js 

echo "setup finished."

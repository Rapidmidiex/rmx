#!/bin/bash
. ./scripts/env.sh
$CONTAINER rm -f $PG_CONTAINER_NAME &> /dev/null || true
$CONTAINER run --name $PG_CONTAINER_NAME -e POSTGRES_PASSWORD=$PG_PASSWORD -p $PG_PORT:5432 -e POSTGRES_USER=$PG_USER -d postgres:14.6-alpine

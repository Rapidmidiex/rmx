#!/bin/bash
. ./scripts/env.sh
. ./scripts/select_container.sh
$CONTAINER rm -f $NATS_CONTAINER_NAME &> /dev/null || true
$CONTAINER run --name $NATS_CONTAINER_NAME -p $NATS_PORT:4222 -d nats:latest -js 
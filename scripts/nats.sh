#!/bin/bash
. ./scripts/select_container.sh
. ./scripts/env.sh
$CONTAINER rm -f $NATS_CONTAINER_NAME &> /dev/null || true
$CONTAINER run --name $NATS_CONTAINER_NAME -p $NATS_PORT:4222 -d nats:latest -js 

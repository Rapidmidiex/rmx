#!/bin/bash
. ./scripts/select_container.sh
. ./scripts/env.sh
$CONTAINER stop $PG_CONTAINER_NAME
$CONTAINER stop $NATS_CONTAINER_NAME
$CONTAINER container rm $PG_CONTAINER_NAME
$CONTAINER container rm $NATS_CONTAINER_NAME
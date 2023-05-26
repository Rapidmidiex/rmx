#!/bin/bash
. ./scripts/select_container.sh
. ./scripts/env.sh
$CONTAINER start $PG_CONTAINER_NAME 
$CONTAINER exec -it $PG_CONTAINER_NAME createdb --username=$PG_USER --owner=$PG_USER $PG_DB

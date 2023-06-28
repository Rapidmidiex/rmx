#!/bin/bash
. ./scripts/env.sh
. ./scripts/select_container.sh
$CONTAINER exec $PG_CONTAINER_NAME createdb --username=$PG_USER --owner=$PG_USER $PG_DB
#!/bin/bash
. ./scripts/select_container.sh
. ./scripts/env.sh
$CONTAINER exec -it $PG_CONTAINER_NAME dropdb --username=$PG_USER $PG_DB

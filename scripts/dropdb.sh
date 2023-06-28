#!/bin/bash
$CONTAINER exec -it $PG_CONTAINER_NAME dropdb --username=$PG_USER $PG_DB

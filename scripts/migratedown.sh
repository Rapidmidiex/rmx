#!/bin/bash
. ./scripts/select_container.sh
. ./scripts/env.sh
migrate -path internal/db/migration -database $PG_CONN_STRING -verbose down

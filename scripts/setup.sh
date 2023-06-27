. ./scripts/select_container.sh &&
. ./scripts/postgres.sh &&
. ./scripts/createdb.sh &&
. ./scripts/migrateup.sh &&
. ./scripts/nats.sh &&

echo "setup finished."

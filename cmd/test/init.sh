#!/bin/bash
set -e

service postgresql start

until pg_isready; do
  echo "Waiting for postgres..."
  sleep 1
done

echo "executing command: $@"

exec "$@"

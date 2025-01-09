#!/bin/bash
set -euo pipefail

cleanup() {
  echo "Cleaning up containers..."
  docker stop mediorum-db 2>/dev/null || true
  docker network rm test-net 2>/dev/null || true
}

trap cleanup EXIT

docker pull postgres:11.4
docker network create test-net 2>/dev/null || true

echo "Starting postgres..."
docker run -d --rm \
  --name mediorum-db \
  --network test-net \
  -e POSTGRES_PASSWORD=example \
  -v "$(pwd)/cmd/mediorum/.initdb:/docker-entrypoint-initdb.d" \
  postgres:11.4

echo "Waiting for postgres..."
until docker exec mediorum-db pg_isready; do sleep 1; done

echo "Building test image..."
docker build \
  --cache-from audius/mediorum-test:latest \
  -t mediorum-test \
  -f cmd/mediorum/Dockerfile.unittests .

echo "Running tests..."
docker run --rm \
  --network test-net \
  -e dbUrlTemplate='postgres://postgres:example@mediorum-db:5432/m%d' \
  -e dbUrl='postgres://postgres:example@mediorum-db:5432/mediorum_test' \
  mediorum-test test

echo "Tests completed successfully"

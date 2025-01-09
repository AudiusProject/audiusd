#!/bin/bash
set -euo pipefail

cleanup() {
  echo "Cleaning up containers..."
  docker stop test-db eth-ganache audiusd-1 audiusd-2 audiusd-3 audiusd-4 2>/dev/null || true
  docker network rm test-net 2>/dev/null || true
}

trap cleanup EXIT

docker network create test-net 2>/dev/null || true
docker pull postgres

echo "Starting postgres..."
docker run -d --rm \
  --name test-db \
  --network test-net \
  -e POSTGRES_PASSWORD=postgres \
  -v "$(pwd)/dev-tools/startup/initdb:/docker-entrypoint-initdb.d" \
  postgres

echo "Waiting for postgres..."
until docker exec test-db pg_isready; do sleep 1; done

echo "Starting eth-ganache..."
docker run -d --rm \
  --name eth-ganache \
  --network test-net \
  -v "$(pwd)/dev-tools:/tmp/dev-tools" \
  audius/eth-ganache:latest \
  bash /tmp/dev-tools/startup/eth-ganache.sh

echo "Building and starting audiusd nodes..."
docker build -t audiusd-test -f ./cmd/audiusd/Dockerfile .
for i in {1..4}; do
  docker run -d --rm \
    --name "audiusd-$i" \
    --network test-net \
    --env-file "./cmd/core/infra/dev_config/content-$i.docker.env" \
    audiusd-test
done

echo "Running tests..."
docker build \
  --cache-from audius/core-test:latest \
  -t core-test \
  -f cmd/core/infra/Dockerfile.tests .

docker run --rm \
  --network test-net \
  core-test test

echo "Tests completed successfully"

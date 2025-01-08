# Development

## Run

Minimal example to run a node and sync it to the audius mainnet.

```bash
docker run --rm -it -p 80:80 audius/audiusd:current

open http://localhost/console/overview
```

## Build

```
make build-audiusd-local

# sync a local node to stage
docker run --rm -it -p 80:80 -e NETWORK=stage  audius/audiusd:$(git rev-parse HEAD)
open http://localhost/console/overview

# network defaults to prod out of box, for an unregistered, RPC mainnet node
docker run --rm -it -p 80:80  audius/audiusd:local
```

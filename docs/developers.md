# Development

## Local Development

First off, add the local x509 cert to your keychain so you can have green ssl in your browser.
> You can skip this, but you will get browser warnings.

```
cd compose/tls
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain compose/tls/dev-cert.pem
```

From the repo root, build and run a local devnet with 4 nodes.

```bash
make build-audiusd-local
make audiusd-dev
```

Access the dev nodes:

```bash
# add -k if you dont have the cert in your keychain
curl https://node1.devnet.audiusd/health-check
curl https://node2.devnet.audiusd/health-check
curl https://node3.devnet.audiusd/health-check
curl https://node4.devnet.audiusd/health-check

# view in browser
open https://node1.devnet.audiusd/console/overview
open https://node2.devnet.audiusd/console/overview
open https://node3.devnet.audiusd/console/overview
open https://node4.devnet.audiusd/console/overview
```

**TODO:** More commands:

```bash
# build a local node
make build-audiusd-local

# test
make build-audiusd-local
make build-audiusd-test
make mediorum-test
make core-test

# sync a locally built node to stage
docker run --rm -it -p 80:80 -p 443:443 -e NETWORK=stage audius/audiusd:local

# sync a locally built node to prod
docker run --rm -it -p 80:80 -p 443:443 audius/audiusd:local

# health check
curl http://localhost/health-check
curl -k https://localhost/health-check

# view in browser
open http://localhost/console/overview
open https://localhost/console/overview
```

## Native Development (macOS)

Build and run audiusd natively on macOS without Docker.

> **FOR HARDCORE AUDIOPHILES ONLY**
> The below may not work exactly as written for you.

### Prerequisites

1. Install system dependencies:
```bash
# Install PostgreSQL
brew install postgresql@15

# Install audio processing dependencies
brew install ffmpeg fftw libsndfile aubio opus libvorbis flac

# Install build tools
brew install go make
```

2. Start PostgreSQL service:
```bash
brew services start postgresql@15
```

### Building

1. Build the binary:
```bash
make bin/audiusd-native
```

2. Create a data directory:
```bash
mkdir -p ~/audiusd/data/postgres
```

3. Initialize the database:
```bash
initdb -D ~/audiusd/data/postgres
createdb audiusd
```

### Running

1. Start audiusd:
```bash
./bin/audiusd-native
```

2. Access the web interface:
```bash
open http://localhost/console/overview
```

### Troubleshooting

- If you get PostgreSQL connection errors, make sure the service is running:
```bash
brew services restart postgresql@15
```

- For audio processing errors, verify all libraries are installed:
```bash
brew list | grep -E 'ffmpeg|fftw|sndfile|aubio|opus|vorbis|flac'
```

- Check logs for detailed error messages:
```bash
tail -f ~/audiusd/logs/audiusd.log
```

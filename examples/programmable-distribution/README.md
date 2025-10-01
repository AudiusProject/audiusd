# Programmable Distribution Example

This example demonstrates geolocation-based content distribution using the OpenAudio protocol. The service uploads a track and provides streaming access only to requests from a specific city (Bozeman, by default).

## How it Works

1. Uploads a demo track to the OpenAudio network
2. Runs an HTTP server that filters stream access by geolocation
3. Returns stream URLs only to requests from the allowed city

## Usage

```bash
go run . -validator node3.audiusd.devnet -port 8080
```

### Flags

- `-validator` - Validator endpoint URL (default: `node3.audiusd.devnet`)
- `-port` - Server port (default: `8080`)

## Testing

Access the streaming endpoint with a city parameter:

```bash
# Allowed city (Bozeman)
curl "http://localhost:8080/stream-access?city=Bozeman"

# Blocked city
curl "http://localhost:8080/stream-access?city=Seattle"
```

## Setup

Copy the demo audio file:

```bash
mkdir -p assets
cp ../../pkg/integration_tests/assets/anxiety-upgrade.mp3 ./assets/
```

## Requirements

- Running Audius validator endpoint

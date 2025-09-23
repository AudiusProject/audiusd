# Programmable Distribution Example

This example demonstrates **Programmable Distribution** - a powerful feature that allows content creators to implement custom logic controlling who can stream their tracks. Track owners can deploy their own filtering services with any access control logic they desire.

## What is Programmable Distribution?

With programmable distribution, the uploader/releaser of an ERN (Entertainment Resource Name) holds the cryptographic keys that define who can stream their content. When they hand out streaming signatures, they can apply any filtering logic:

- **Geolocation-based**: Only allow streaming from specific cities or regions
- **Time-based**: Restrict access to certain hours or dates
- **Payment-gated**: Require payment or subscription verification
- **Token-gated**: Check for NFT ownership or token balances
- **Custom rules**: Any logic you can imagine!

## How It Works

1. **Upload Track**: The content creator uploads their audio file to the network via mediorum
2. **Create ERN**: An ERN is created with the transcoded audio CID, establishing ownership
3. **Deploy Filter Service**: The creator runs a service that validates streaming requests
4. **Control Access**: When users request streaming URLs, the filter service decides who gets access

## Architecture

```
User Request → Filter Service → Validation Logic → Stream Access
                     ↓                                    ↓
              (Your Custom Code)                   (Signed URLs)
```

## Example Implementation

This example implements geolocation-based filtering that only allows streaming from Bozeman, Montana.

### Files

- `main.go` - Main entry point, initializes SDK and starts the server
- `handler.go` - HTTP handler implementing geolocation filtering logic
- `upload.go` - Handles file upload to mediorum and ERN creation

### Key Components

#### 1. File Upload and Transcoding
```go
// Upload to mediorum and get transcoded CID
uploads, err := auds.Mediorum.UploadFile(ctx, audioFile, "anxiety-upgrade.mp3", uploadOpts)
transcodedCID := upload.GetTranscodedCID()
```

#### 2. ERN Creation
```go
// Create ERN with the transcoded CID
ernMessage := &ddexv1beta1.NewReleaseMessage{
    // ... ERN metadata ...
    ResourceList: []*ddexv1beta1.Resource{
        // Resource with transcoded CID
    }
}
```

#### 3. Access Control Filter
```go
// Check if request is from allowed city
if !strings.EqualFold(city, h.allowedCity) {
    return 403 // Access denied
}

// Generate streaming signature for approved requests
sig := &v1.StreamERNSignature{
    Addresses: addresses,
    ExpiresAt: timestamppb.New(expiry),
}
```

## Running the Example

### Prerequisites
- Go 1.19 or higher
- Access to an Audius node (default: validator.audius.co)
- An audio file at `./assets/anxiety-upgrade.mp3` (optional)

### Steps

1. **Run the example**:
```bash
go run .
```

2. **Test access control**:

Allowed (from Bozeman):
```bash
curl 'http://localhost:8080/stream-access?city=Bozeman'
```

Blocked (from other cities):
```bash
curl 'http://localhost:8080/stream-access?city=Seattle'
```

## Customizing the Filter Logic

To implement your own distribution logic, modify the `ServeHTTP` method in `handler.go`:

```go
func (h *GeolocationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Add your custom validation logic here
    // Examples:
    // - Check payment status
    // - Verify NFT ownership
    // - Validate time windows
    // - Check user reputation

    // If approved, generate streaming signature
    // If denied, return error
}
```

## Production Considerations

For production deployments:

1. **Deploy to Edge**: Use Cloudflare Workers, Vercel Edge Functions, or similar for global low-latency
2. **Add Authentication**: Implement proper user authentication and session management
3. **Rate Limiting**: Protect against abuse with rate limiting
4. **Monitoring**: Track usage patterns and access attempts
5. **Caching**: Cache validation results for better performance

## Example Deployment Platforms

### Cloudflare Workers
```javascript
export default {
  async fetch(req, env) {
    const { city } = req.cf || {}
    if (city !== "Bozeman") {
      return Response.json({ error: "Access denied" }, { status: 403 })
    }
    // Generate signature...
  }
}
```

### AWS Lambda
Deploy the Go handler directly to Lambda with API Gateway

### Vercel Edge Functions
Use the Vercel Go runtime for edge deployment

## Benefits

- **Full Control**: Content creators maintain complete control over distribution
- **Flexible Monetization**: Implement any payment or access model
- **Privacy**: Access control logic remains private to the creator
- **Composability**: Combine multiple filtering strategies
- **Decentralized**: No central authority controls access

## Learn More

- [Audius Protocol Documentation](https://docs.audius.org)
- [ERN Specification](https://ddex.net/standards/)
- [SDK Documentation](https://github.com/AudiusProject/audiusd)

## License

This example is provided for educational purposes. See the main repository for license details.
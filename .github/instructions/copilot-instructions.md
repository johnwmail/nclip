# Nclip Copilot Instructions

This is a Go-based HTTP clipboard/pastebin service using the Gin framework. The service supports two modes:
- **Lambda mode:** Content is stored in S3 as objects (`$slug`), with metadata in a JSON file (`$slug.json`).
- **Server mode:** Content is stored in the local filesystem as files (`$slug`), with metadata in a JSON file (`$slug.json`).

## Core Architecture

- **Framework**: Gin HTTP router
**Storage**: Abstracted behind `PasteStore` interface
    - Filesystem implementation for server mode
    - S3 implementation for AWS Lambda
- **Data Format**: JSON metadata + raw binary content
- **Configuration**: Environment variables + CLI flags

## API Endpoints

### Core Endpoints
- `GET /` — Web UI (upload form, stats)
- `POST /` — Upload paste (returns URL)
- `POST /burn/` — Create burn-after-read paste
- `GET /{slug}` — HTML view of paste
- `GET /raw/{slug}` — Raw content download
- `GET /api/v1/meta/{slug}` — JSON metadata (no content)
- `GET /json/{slug}` — Alias for `/api/v1/meta/{slug}` (shortcut)

### System Endpoints
- `GET /health` — Health check (200 OK)

## Data Models

### Paste Metadata
```go
type Paste struct {
    ID            string     `json:"id"`
    CreatedAt     time.Time  `json:"created_at"`
    ExpiresAt     *time.Time `json:"expires_at,omitempty"`
    Size          int64      `json:"size"`
    ContentType   string     `json:"content_type"`
    BurnAfterRead bool       `json:"burn_after_read"`
    ReadCount     int        `json:"read_count"`
}
```

### Configuration
```go
type Config struct {
    Port           int           `default:"8080"`
    URL            string        `default:""`
    SlugLength     int           `default:"5"`
    BufferSize     int64         `default:"5242880"` // 5MB
    DefaultTTL     time.Duration `default:"24h"`
    S3Bucket       string        `default:""` // S3 bucket for Lambda mode
}
```

## Storage Interface
- For storage backends, server mode uses the filesystem, and AWS Lambda uses S3. Both implementations should adhere to the same interface. No storage environment variable is needed.

```go
type PasteStore interface {
    Store(paste *Paste) error
    Get(id string) (*Paste, error)
    Delete(id string) error
    IncrementReadCount(id string) error
}
```

## Implementation Requirements

### Input Handling
- Accept raw POST data for text/binary content
- Auto-detect content type from file extension or content
- Generate random slug IDs (configurable length)

### TTL/Expiration
- Default 24-hour expiration (configurable)
- Expiry is handled by application logic (no DB TTL indexes)

### Burn After Read
- Mark pastes as `burn_after_read: true`
- Delete immediately after first read via `GET /{slug}`
- Raw access `/raw/{slug}` also triggers burn

### Error Handling
- Standard JSON error format: `{"error": "message"}`
- HTTP status codes: 404 for not found, 500 for server errors
- Graceful degradation when storage is unavailable

### Security
- No authentication required
- No rate limiting (keep simple)
- Validate slug format (alphanumeric only)
- Limit upload size via `BufferSize` config

### Logging
- Structured logging (JSON format recommended)
- Optional tracing for debugging

### Web UI
- Simple HTML form for paste upload
- Display paste URL after upload
- Show paste statistics (size, type, etc.)
- Raw/download buttons for existing pastes
- Responsive design for mobile

### CLI Compatibility
- `curl --data-binary @- http://host/` — Upload from stdin
- `curl --data-binary @file http://host/` — Upload file
- Return plain text URLs for CLI usage
- Content-Type detection for proper display

## Code Organization

```
/
├── main.go              # Entry point, config, router setup
├── config/
│   └── config.go        # Configuration struct and parsing
├── storage/
│   ├── interface.go     # PasteStore interface
│   ├── filesystem.go    # Filesystem implementation (server mode)
│   └── s3.go            # S3 implementation (Lambda mode)
├── handlers/
│   ├── paste.go         # Upload, view, raw endpoints
│   ├── burn.go          # Burn-after-read functionality
│   ├── meta.go          # Metadata endpoint
│   └── system.go        # Health endpoint
├── models/
│   └── paste.go         # Paste struct and utilities
├── static/
│   ├── index.html       # Web UI
│   ├── style.css        # Styling
│   └── script.js        # Frontend JS
└── utils/
    ├── slug.go          # Slug generation
    └── mime.go          # Content-type detection
```

## Testing Strategy

- Unit tests for all storage implementations
- HTTP endpoint tests using httptest
- Mock PasteStore for handler testing
- Integration tests with real storage backends
- Test both Lambda and server deployment modes

## Environment Variables

All config via env vars with CLI flag alternatives:
- `NCLIP_PORT` / `--port` (default: 8080) — HTTP server port
- `NCLIP_URL` / `--url` (default: auto-detect) — Base URL for paste links (e.g. "https://demo.nclip.app")
- `NCLIP_SLUG_LENGTH` / `--slug-length` (default: 5) — Length of generated paste IDs
- `NCLIP_BUFFER_SIZE` / `--buffer-size` (default: 5242880) — Max upload size in bytes (5MB)
- `NCLIP_TTL` / `--ttl` (default: "24h") — Default paste expiration time


**Note**: Storage backend is automatically selected based on deployment environment:
- Server mode: Uses filesystem
- AWS Lambda: Uses S3 (NCLIP_S3_BUCKET)
- Detection via AWS_LAMBDA_FUNCTION_NAME environment variable


## Deployment Modes

1. **Server mode**: Uses filesystem, full HTTP server
2. **AWS Lambda**: Uses S3, same codebase with Lambda adapter
3. **Both Lambda and server mode should use the same main.go, and use the AWS_LAMBDA_FUNCTION_NAME variable to determine the mode. (Don't separate 2 main.go for 2 modes )**

Both modes share identical API and behavior.

## Lambda Deployment with S3
- The Lambda code should support http v2.0 payload format. (This is the default for new Lambda functions behind an API Gateway HTTP API.)
- Use AWS Lambda Go SDK
- Wrap Gin router with Lambda adapter
- S3 bucket is specified via `NCLIP_S3_BUCKET` env var
- Ensure the Lambda execution role has permissions for S3 actions: `s3:GetObject`, `s3:PutObject`, `s3:DeleteObject`

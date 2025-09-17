# nclip Copilot Instructions

This is a Go-based HTTP clipboard/pastebin service using the Gin framework. The service supports both MongoDB (container/on-prem) and DynamoDB (AWS Lambda) storage backends.

## Core Architecture

- **Framework**: Gin HTTP router
- **Storage**: Abstracted behind `PasteStore` interface
  - MongoDB implementation for container/K8s deployment
  - DynamoDB implementation for AWS Lambda
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

### System Endpoints
- `GET /health` — Health check (200 OK)
- `GET /metrics` — Prometheus metrics (optional)

## Data Models

### Paste Metadata
```go
type Paste struct {
    ID           string    `json:"id" bson:"_id"`
    CreatedAt    time.Time `json:"created_at" bson:"created_at"`
    ExpiresAt    *time.Time `json:"expires_at" bson:"expires_at,omitempty"`
    Size         int64     `json:"size" bson:"size"`
    ContentType  string    `json:"content_type" bson:"content_type"`
    BurnAfterRead bool     `json:"burn_after_read" bson:"burn_after_read"`
    ReadCount    int       `json:"read_count" bson:"read_count"`
    Content      []byte    `json:"-" bson:"content"` // Not exposed in JSON
}
```

### Configuration
```go
type Config struct {
    Port           int    `default:"8080"`
    URL            string `default:""`
    SlugLength     int    `default:"5"`
    BufferSize     int64  `default:"1048576"` // 1MB
    DefaultTTL     time.Duration `default:"24h"`
    EnableMetrics  bool   `default:"true"`
    EnableWebUI    bool   `default:"true"`
    StorageType    string `default:"mongodb"` // "mongodb" or "dynamodb"
    MongoURL       string `default:"mongodb://localhost:27017"`
    DynamoTable    string `default:"nclip-pastes"`
}
```

## Storage Interface

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
- Support multipart form uploads for files
- Auto-detect content type from file extension or content
- Generate random slug IDs (configurable length)

### TTL/Expiration
- Default 24-hour expiration (configurable)
- MongoDB: Use TTL indexes on `expires_at`
- DynamoDB: Use TTL attribute

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

### Logging & Metrics
- Structured logging (JSON format recommended)
- Prometheus metrics for requests, errors, paste counts
- Optional tracing for debugging

### Web UI
- Simple HTML form for paste upload
- Display paste URL after upload
- Show paste statistics (size, type, etc.)
- Raw/download buttons for existing pastes
- Responsive design for mobile

### CLI Compatibility
- `curl -d @- http://host/` — Upload from stdin
- `curl -F 'file=@path' http://host/` — Upload file
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
│   ├── mongodb.go       # MongoDB implementation
│   └── dynamodb.go      # DynamoDB implementation
├── handlers/
│   ├── paste.go         # Upload, view, raw endpoints
│   ├── burn.go          # Burn-after-read functionality
│   ├── meta.go          # Metadata endpoint
│   └── system.go        # Health, metrics endpoints
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
- Integration tests with real databases
- Test both Lambda and container deployment modes

## Environment Variables

All config via env vars with CLI flag alternatives:
- `NCLIP_PORT` / `--port`
- `NCLIP_URL` / `--url`
- `NCLIP_SLUG_LENGTH` / `--slug-length`
- `NCLIP_BUFFER_SIZE` / `--buffer-size`
- `NCLIP_TTL` / `--ttl`
- `NCLIP_ENABLE_METRICS` / `--enable-metrics`
- `NCLIP_ENABLE_WEBUI` / `--enable-webui`
- `NCLIP_STORAGE_TYPE` / `--storage-type`
- `NCLIP_MONGO_URL` / `--mongo-url`
- `NCLIP_DYNAMO_TABLE` / `--dynamo-table`

## Deployment Modes

1. **Container/K8s**: Uses MongoDB, full HTTP server
2. **AWS Lambda**: Uses DynamoDB, same codebase with Lambda adapter

Both modes share identical API and behavior.
# Nclip Copilot Instructions

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
- `GET /json/{slug}` — Alias for `/api/v1/meta/{slug}` (shortcut)

### System Endpoints
- `GET /health` — Health check (200 OK)

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
    MongoURL       string `default:"mongodb://localhost:27017"`
    DynamoTable    string `default:"nclip-pastes"`
}
```

## Storage Interface
- For Storage backends, Container/K8s must uses MongoDB, and AWS Lambda must uses DynamoDB. Both implementations should adhere to the same interface. So, no Storage Environment variable in neededs.

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
│   ├── mongodb.go       # MongoDB implementation
│   └── dynamodb.go      # DynamoDB implementation
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
- Integration tests with real databases
- Test both Lambda and container deployment modes

## Environment Variables

All config via env vars with CLI flag alternatives:
- `NCLIP_PORT` / `--port` (default: 8080) — HTTP server port
- `NCLIP_URL` / `--url` (default: auto-detect) — Base URL for paste links (e.g. "https://demo.nclip.app")
- `NCLIP_SLUG_LENGTH` / `--slug-length` (default: 5) — Length of generated paste IDs
- `NCLIP_BUFFER_SIZE` / `--buffer-size` (default: 1048576) — Max upload size in bytes (1MB)
- `NCLIP_TTL` / `--ttl` (default: "24h") — Default paste expiration time
- `NCLIP_MONGO_URL` / `--mongo-url` (default: "mongodb://localhost:27017") — MongoDB connection URL (container mode)
- `NCLIP_DYNAMO_TABLE` / `--dynamo-table` (default: "nclip-pastes") — DynamoDB table name (Lambda mode)

**Note**: Storage backend is automatically selected based on deployment environment:
- Container/K8s: Uses MongoDB (NCLIP_MONGO_URL)
- AWS Lambda: Uses DynamoDB (NCLIP_DYNAMO_TABLE)
- Detection via AWS_LAMBDA_FUNCTION_NAME environment variable

## Deployment Modes

1. **Container/K8s**: Uses MongoDB, full HTTP server
2. **AWS Lambda**: Uses DynamoDB, same codebase with Lambda adapter
3. **Both Lambda and Container, should be use the same main.go, and use the AWS_LAMBDA_FUNCTION_NAME variable to determine the mode. (Don't separate 2 main.go for 2 modes )**

Both modes share identical API and behavior.

## Lambda Deployment with DynamoDB
- The Lambda code should supprt http v2.0 payload format. (This is the default for new Lambda functions behind an API Gateway HTTP API.)
- Both Lambda 
- Use AWS Lambda Go SDK
- Wrap Gin router with Lambda adapter
- The DynamoDB table must have a primary key `id` (string) and a TTL attribute `expires_at` (number, epoch time).
- The DynamoDB region should match the Lambda region.
- The DynamoDB table name is configurable via `NCLIP_DYNAMO_TABLE` env var (default `nclip-pastes`).
- Ensure the Lambda execution role has permissions for DynamoDB actions: `dynamodb:GetItem`, `dynamodb:PutItem`, `dynamodb:DeleteItem`, `dynamodb:UpdateItem`, and `dynamodb:DescribeTable`.

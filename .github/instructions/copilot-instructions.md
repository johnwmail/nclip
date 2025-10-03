# Nclip Copilot Instructions

Nclip is a Go-based HTTP clipboard/pastebin service built with Gin that supports dual deployment modes: **server mode** (filesystem storage) and **Lambda mode** (S3 storage).

## Essential Architecture Knowledge

### Deployment Mode Detection
The codebase automatically detects deployment mode via `isLambdaEnvironment()` checking for `AWS_LAMBDA_FUNCTION_NAME` env var. This drives storage backend selection:
- **Server mode**: Uses `storage.NewFileSystemStore("./data")` 
- **Lambda mode**: Uses `storage.NewS3Store(cfg.S3Bucket, cfg.S3Prefix)`

### Storage Abstraction Pattern
All storage operations go through the `PasteStore` interface:
```go
type PasteStore interface {
    Store(paste *models.Paste) error
    Get(id string) (*models.Paste, error) 
    Exists(id string) (bool, error)
    Delete(id string) error
    IncrementReadCount(id string) error
    StoreContent(id string, content []byte) error
    GetContent(id string) ([]byte, error)
}
```

Key insight: Metadata and content are stored separately. Both filesystem and S3 implementations store:
- Content as raw bytes (`{slug}` file/object)
- Metadata as JSON (`{slug}.json` file/object)

### Handler Architecture 
Handlers are organized by function, not REST resources:
- `handlers/paste.go` - Core upload/retrieval logic
- `handlers/webui.go` - HTML UI endpoints  
- `handlers/meta.go` - Metadata endpoints
- `handlers/system.go` - Health/status endpoints
- `handlers/upload/` and `handlers/retrieval/` - Specialized logic

## Critical Implementation Patterns

### Slug Generation with Collision Handling
Uses batch generation with fallback to longer slugs in `generateUniqueSlug()`:
```go
lengths := []int{5, 6, 7}  // Try incrementally longer slugs
candidates, err := utils.GenerateSlugBatch(batchSize, length)
```
Also checks if existing slugs are expired before considering them collisions.

### Content-Type Detection Chain
1. Use client-provided `Content-Type` header if present
2. Detect from filename extension (if filename provided)  
3. Fall back to content-based detection via `utils/mime.go`

### Burn-After-Read Implementation
Critical: Burn happens on **any content access** (`GET /{slug}` or `GET /raw/{slug}`), not just metadata access. Uses atomic read-then-delete pattern.

### Error Response Consistency 
The codebase has specific requirements for 404 handling:
- **CLI clients** (curl/wget): Return JSON `{"error": "message"}` 
- **Browser clients**: Render HTML error page using `view.html` template
- Detection via User-Agent string analysis

## Testing & Development Workflows

### Test Cleanup Requirements ⚠️
**CRITICAL**: All tests must clean up artifacts they create. The integration test script (`scripts/integration-test.sh`) uses:
- `TRASH_RECORD_FILE="/tmp/nclip_integration_slugs.txt"` to track created slugs
- Cleanup function removes only recorded slugs or recently modified files (`-mmin -60`)
- Never use broad cleanup like `rm -rf ./data/*` - it may delete unrelated data

### Build Commands
- Development: `go run .` 
- Docker: Multi-stage build with version injection via `--build-arg`
- Integration tests: `./scripts/integration-test.sh` (requires running server)

### Environment Variables
Key config vars (all prefixed with `NCLIP_`):
- `NCLIP_DATA_DIR` - Storage directory for server mode (default: "./data")
- `NCLIP_PORT`, `NCLIP_TTL`, `NCLIP_BUFFER_SIZE` - Basic config
- `NCLIP_S3_BUCKET`, `NCLIP_S3_PREFIX` - Lambda mode S3 settings
- `DEBUG` - Enables verbose logging including all environment variables

## Deployment Specifics

### Docker/Kubernetes
- Uses non-root user (1001:1001) 
- Read-only container filesystem
- Health check via `/health` endpoint
- Static assets copied to `/app/static/`

### Lambda Integration
- Uses `awslabs/aws-lambda-go-api-proxy` for Gin integration
- Gin routes remain identical between server and Lambda modes
- Lambda handler wraps existing Gin router - no code duplication

### Version Management
Build-time injection pattern:
```bash
go build -ldflags="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.CommitHash=$GIT_COMMIT"
```

This architecture enables true "write once, deploy anywhere" with the same codebase running in containers, servers, and serverless environments.


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

### Buffer Size Limit Implementation
**CRITICAL**: Buffer size limits must be properly enforced to prevent silent truncation. The correct pattern handles multipart vs direct uploads differently:

```go
// Check if this is a multipart upload
isMultipart := strings.HasPrefix(c.ContentType(), "multipart/form-data")

// For direct POST requests, check Content-Length early (accurate for content size)
// Skip for multipart since Content-Length includes boundaries and headers
if !isMultipart {
    if contentLength := c.Request.ContentLength; contentLength > 0 && contentLength > bufferSize {
        return error
    }
}

// Read with io.LimitReader
content, err := io.ReadAll(io.LimitReader(reader, bufferSize))

// Always check for truncation by attempting to read one more byte
var oneByte [1]byte
n, _ := reader.Read(oneByte[:])
if n > 0 {
    return error // Content was truncated
}
```

This prevents silent truncation while being accurate for both direct POST and multipart file uploads.

### Platform-Specific Buffer Size Limits ⚠️
**CRITICAL**: When deploying Nclip, consider platform-specific limits beyond the application-level `NCLIP_BUFFER_SIZE`. See deployment-specific documentation for details:
- **Lambda deployments**: Refer to `Documents/LAMBDA.md` for AWS Lambda 6MB payload limits
- **API Gateway/CloudFront**: Check AWS service limits for your architecture

### Error Response Consistency 
The codebase has specific requirements for 404 handling:
- **CLI clients** (curl/wget): Return JSON `{"error": "message"}` 
- **Browser clients**: Render HTML error page using `view.html` template
- Detection via User-Agent string analysis

## Testing & Development Workflows

### Required Before Each Commit
- Run `go fmt ./...` and `golangci-lint run` before committing any changes to ensure proper code formatting and linting
- This will run gofmt on all Go files to maintain consistent style

### Development Flow
- Code changes, add new feature and bug fixes
  ** If adding new features, add corresponding unit tests, integration tests, and update documentation (Documents/* and README.md as needed)
- Ensure no linting errors with `golangci-lint run`, `go fmt ./...`, and `go vet ./...`
- Run unit tests with `go test ./...` to verify functionality
- Run integration tests with `bash scripts/integration-test.sh` (requires stop and re-running server, `pkill nclip` and `go run . &`)
- Address any issues found during testing and repeat until all tests pass
- Before pushing changes, ensure all above steps is passed and code is clean

### Test Cleanup Requirements ⚠️
**CRITICAL**: All tests must clean up artifacts they create. The unified integration test script (`scripts/integration-test.sh`) uses:
- `TRASH_RECORD_FILE="/tmp/nclip_integration_slugs.txt"` to track created slugs
- Cleanup function removes only recorded slugs or recently modified files (`-mmin -60`)
- Never use broad cleanup like `rm -rf ./data/*` - it may delete unrelated data
- Cleanup runs automatically via EXIT trap handlers in lib.sh
- Unit tests use `defer cleanupTestData(store.dataDir)` to remove test files

This requirement ensures reproducible, reliable tests and prevents leftover artifacts from affecting subsequent runs or deployments.

### Buffer Size Testing
Buffer size limits are tested at multiple levels:
- **Unit tests**: `TestBufferSizeLimit` tests both direct POST and multipart uploads
- **Integration tests**: `test_buffer.sh` module in unified test suite tests end-to-end with real HTTP requests
- Both test that oversized uploads are rejected with 400 status and appropriate error messages

### Upload-auth testing
Upload-auth is a runtime toggle that requires clients to present an API key to upload content. Important points for tests and CI:
- `NCLIP_UPLOAD_AUTH` (true/false) enables or disables upload API key enforcement.
- `NCLIP_API_KEYS` contains the comma-separated allowed keys the server accepts. In CI the integration job generates a random key and writes it here.
- `NCLIP_TEST_API_KEY` (or `NCLIP_TEST_API_KEY_BEARER`) is used by the integration script to authenticate requests when `NCLIP_UPLOAD_AUTH=true`.
- Integration tests assert unauthenticated uploads return 401 when auth is enabled and verify authenticated uploads succeed using the generated test key.

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


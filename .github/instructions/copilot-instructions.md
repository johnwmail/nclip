## Test and Script Cleanup Requirement

All integration tests, scripts, and automated test routines must clean up any files, directories, or artifacts they create. This includes:
- Removing all files in the data directory (e.g., ./data/*) created during tests
- Deleting any temporary files (e.g., /tmp/nclip_test.zip) or injected test artifacts
- Ensuring the environment is clean before and after every test run

This requirement ensures reproducible, reliable tests and prevents leftover artifacts from affecting subsequent runs or deployments.
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
- `curl -sL --data-binary @- http://host/` — Upload from stdin
- `curl -sL --data-binary @file http://host/` — Upload file
- Return plain text URLs for CLI usage
- Content-Type detection for proper display

## Code Organization

```
## nclip Copilot Instructions (2025)

This document describes the current architecture, coding conventions, testing and deployment practices for `nclip`. It's targeted at contributors and any automated copilot/coding agent assisting with the codebase.

### 1) Test and Script Cleanup (must follow)

All integration tests and scripts must clean up the artifacts they create. Recommended patterns:
- Create test artifacts with a predictable prefix (e.g. `nclip-test-<slug>`), and delete only matching files during cleanup.
- When prefixing isn't possible, delete files in `./data` only if they are recently modified (e.g. `-mmin -60`) and/or explicitly recorded in a temp file during the test run.
- Always remove temporary files in `/tmp/` created by tests (use explicit names).

This avoids accidental deletion of unrelated data and keeps CI reproducible.

---

### 2) High-level Architecture

- Language: Go (1.25+ recommended)
- HTTP framework: Gin
- Storage abstraction: `PasteStore` interface (Filesystem and S3 implementations)
- Data model: JSON metadata + raw binary content
- Configuration: environment variables and CLI flags

### 3) API Endpoints (current)

- GET / — Web UI (upload form)
- POST / — Upload paste (returns paste URL)
- POST /burn/ — Burn-after-read paste
- GET /{slug} — HTML view
- GET /raw/{slug} — Raw content download (sets Content-Disposition)
- GET /api/v1/meta/{slug} — Metadata JSON
- GET /json/{slug} — Alias for metadata
- GET /health — Health check

### 4) Data Types

Paste metadata (canonical):

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

### 5) Storage interface

```go
type PasteStore interface {
    StoreContent(id string, content []byte) error
    StoreMetadata(id string, meta *Paste) error
    GetMetadata(id string) (*Paste, error)
    GetContent(id string) ([]byte, error)
    Delete(id string) error
    IncrementReadCount(id string) error
}
```

Note: The concrete methods in this repository follow this pattern (filesystem uses files, S3 uses object + metadata JSON files).

### 6) Implementation notes

- Content-Type: Prefer client-provided `Content-Type` header; fall back to filename extension and then content-based detection. Keep `utils/mime.go` small and testable.
- Filenames: Download endpoints should include a user-friendly extension (via `utils.ExtensionByMime`). Text content should be served inline; binaries as attachments.
- Burn-after-read: Ensure atomic delete after the first successful read (store-level delete or transactional approach).
- Slugs: Uppercase alphanumeric by default; configurable length.

### 7) Error handling and logging

- Return JSON errors: `{ "error": "message" }`
- Use appropriate HTTP status codes
- Prefer structured logs (key/value or JSON); guard verbose debug behind a debug flag or environment variable

#### Consistent NotFound behavior (important)

- Non-existing slug and a second access to a burn-after-read paste MUST return the same response semantics.
- For CLI/API clients (detected via User-Agent like `curl`, `wget`): return HTTP 404 with JSON body: `{ "error": "<meaningful message>" }`.
- For web browser UI clients: return HTTP 404 and render the HTML `view.html` page with a friendly, prominent message in the UI (e.g. "Paste not available — it may have been deleted or already burned after reading.").
- The server-side handlers should centralize this behavior (use a helper) so all not-found cases use the same message and status code.

This ensures a consistent developer experience for CLI and a clear UX for browser users.

### 8) Testing

- Unit tests for utils, storage, and services
- Handler tests using httptest and a Mock PasteStore
- Integration tests (scripts/integration-test.sh) exercise the real binary and filesystem backend
- CI runs: unit tests, `golangci-lint` (includes `gocyclo`), and integration tests on main/dev branches

### 9) CI / Linting

- `golangci-lint` is used (configurable via `.golangci.yml`). `gocyclo` is enabled; keep functions under complexity thresholds where practical. Refactor complex helpers into small, testable functions.

### 10) Deployment

- Server mode: standard HTTP server with filesystem storage
- Lambda mode: S3-backed, same codebase. Use `AWS_LAMBDA_FUNCTION_NAME` or environment-driven adapter to detect Lambda runtime. Wrap Gin with an adapter (aws-lambda-go-api-proxy or similar).

### 11) Security and operational notes

- No authentication required by design; consider adding rate-limiting or abuse protection for public deployments.
- Ensure S3 permissions are scoped to required actions only.

---

If you want, I can also:
- Convert integration-test cleanup to a prefix-based approach, or
- Add a short contributor checklist in this file with exact commands to run locally (build, run server, run integration tests).

If you'd like this file split into separate CONTRIBUTING.md and ARCHITECTURE.md files I can create them as well.

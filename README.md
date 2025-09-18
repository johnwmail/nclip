

# nclip Gin Refactor: Requirements & API

- Use Gin for all HTTP endpoints, supporting both AWS Lambda (DynamoDB) and container/on-prem (MongoDB) deployments.
- Share the same codebase, logic, UI, and data format for both environments.
- Accept raw/binary input for pastes (text or file) via curl or web UI.
- Keep MongoDB and DynamoDB data formats as similar as possible.



## API Endpoints

- `GET /` — Web UI (form for upload, stats, etc.)
- `POST /` — Upload new paste (raw or file data; returns paste URL)
- `POST /burn/` — Create burn-after-read paste (deleted after first read)
- `GET /$slug` — HTML view of paste (browser)
- `GET /raw/$slug` — Raw data (text or binary, for curl/cli/download)
- `GET /api/v1/meta/$slug` — JSON metadata (see below)
- `GET /json/$slug` — Alias for `/api/v1/meta/$slug` (shortcut)
- `GET /health` — Health check (200 OK)
- `GET /metrics` — Prometheus metrics (can be disabled)


## Paste Metadata (JSON)

Returned by `GET /api/v1/meta/$slug` or `GET /json/$slug`. Does **not** include the actual content.

```json
{
  "id": "string",                  // Unique paste ID
  "created_at": "2025-09-17T12:34:56Z", // ISO8601 timestamp
  "expires_at": "2025-09-18T12:34:56Z", // ISO8601 (optional, null if no expiry)
  "size": 12345,                    // Size in bytes
  "content_type": "text/plain",    // MIME type
  "burn_after_read": true,          // true if burn-after-read
  "read_count": 0                   // Number of times read (optional)
}
```

*Access content via `/raw/$slug` or `/$slug`, not via metadata.*

## Usage Examples

```bash
# Upload a paste
echo "hello world" | curl --data-binary @- http://localhost:8080/
# Returns: http://localhost:8080/abc123

# Get the content
curl http://localhost:8080/abc123
curl http://localhost:8080/raw/abc123

# Get metadata (JSON)
curl http://localhost:8080/api/v1/meta/abc123
curl http://localhost:8080/json/abc123  # Shortcut alias
```


**Usage Notes:**
- Use `POST /burn/` to create a burn-after-read paste (deleted after first read via `GET /$slug`).
- All uploads/downloads use raw/binary data (not JSON) for maximum compatibility with curl and file uploads.
- JSON is only for error responses and metadata endpoints.
- Web UI and API both support burn-after-reading and raw/download features.


## Best Practices

- Abstract storage behind a `PasteStore` interface (MongoDB/DynamoDB).
- Unit tests must cover both storage backends (use mocks as needed).
- All endpoints should return a standard JSON error format: `{ "error": "message" }`.
- Reserve `/api/v1/` as a prefix for future API versioning.
- Provide OpenAPI/Swagger or markdown docs for all endpoints.
- Use structured logging and Prometheus metrics (optionally tracing).
- Support graceful shutdown (SIGTERM/SIGINT) in server mode.
- All features must pass in CI.

---



## Implementation Checklist

- HTTP only (no TCP), default port 8080
- DynamoDB storage for Lambda; MongoDB for container/K8s
- Auto-expiration (TTL), default 1 day (configurable)
- Domain/host detected from HTTP header, override with `NCLIP_URL`

- Health check: `/health`
- Prometheus metrics: `/metrics` (can disable with `NCLIP_ENABLE_METRICS=false`)
- Web UI at `/` (can disable with `NCLIP_ENABLE_WEBUI=false`)
- Web UI: form for upload, shows paste URL, stats, raw/download buttons
- Burn after reading: `POST /burn/` to create, `GET /$slug` to fetch/delete
- Raw output: `/raw/$slug`
- Slug length: default 5 (configurable via `NCLIP_SLUG_LENGTH`)
- Max buffer size: 1MB (configurable via `NCLIP_BUFFER_SIZE`)
- All config via env vars and matching CLI flags (e.g., `NCLIP_URL`/`--url`)
- Single binary, no external deps except MongoDB/DynamoDB
- Shared logic/code for both Lambda and container; only storage differs
- Cleanup: remove obsolete/unused files after refactor
- Code must pass `go fmt`, `go vet`, `golangci-lint`, and have good test coverage


[![Test](https://github.com/johnwmail/nclip/workflows/Test/badge.svg)](https://github.com/johnwmail/nclip/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/johnwmail/nclip)](https://goreportcard.com/report/github.com/johnwmail/nclip)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub release](https://img.shields.io/github/release/johnwmail/nclip.svg)](https://github.com/johnwmail/nclip/releases)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org/)


# nclip


A modern, high-performance net-to-clipboard service written in Go, inspired by [fiche](https://github.com/solusipse/fiche).


## Overview


nclip is an HTTP clipboard service that accepts content via:
- **HTTP/curl** - Modern web API: `echo "text" | curl --data-binary @- http://localhost:8080`
- **HTTP/curl** - Web API with multiline support: `ps | curl --data-binary @- http://localhost:8080`
- **Web UI** - Browser interface at `http://localhost:8080`
- **File upload** - Upload files via web UI or curl: `curl --data-binary @/path/to/file http://localhost:8080`
- **Raw access** - Access raw content via `http://localhost:8080/raw/SLUG` or `http://localhost:8080/SLUG?raw=true`
- **Burn after reading** - Content that self-destructs after being accessed once via `http://localhost:8080/SLUG?burn=true` or `http://localhost:8080/burn/SLUG`

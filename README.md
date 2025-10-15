[![Test](https://github.com/johnwmail/nclip/workflows/Test/badge.svg)](https://github.com/johnwmail/nclip/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/johnwmail/nclip)](https://goreportcard.com/report/github.com/johnwmail/nclip)
[![codecov](https://codecov.io/gh/johnwmail/nclip/branch/main/graph/badge.svg?token=G9K6YJH1XK)](https://codecov.io/gh/johnwmail/nclip)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub release](https://img.shields.io/github/release/johnwmail/nclip.svg)](https://github.com/johnwmail/nclip/releases)
[![Go Version](https://img.shields.io/badge/go-1.23+-blue.svg)](https://golang.org/)

# NCLIP

A modern, high-performance HTTP clipboard app written in Go with Gin framework.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Quick Start](#quick-start)
- [Deployment](#deployment)
- [Configuration](#configuration)
- [API Endpoints](#api-endpoints)
- [Development](#development)
- [Links](#links)

<a id="overview"></a>
## Overview

nclip is a versatile clipboard app that accepts content via:
- **Web UI** - Browser interface at `http://localhost:8080`
- **Curl** - Modern web API: `echo "text" | curl -sL --data-binary @- http://localhost:8080`
- **File upload** - Upload (small) files via web UI or curl: `curl -sL --data-binary @/path/file http://localhost:8080`
- **Raw access** - Access raw content via `http://localhost:8080/raw/SLUG`
- **Burn after reading** - Content that self-destructs after being accessed once

<a id="features"></a>
## ✨ Features

- 🚀 **Dual Deployment**: Server mode (local or container) + AWS Lambda
- 🎯 **Unified Codebase**: Same code, logic, and UI for both environments
- 🗄️ **Multi-Storage Backend**: Filesystem for server mode, S3 for Lambda
- 🐳 **Container Ready**: Docker & Kubernetes deployment
- ⏰ **Auto-Expiration**: TTL support with configurable defaults
- 🛡️ **Production Ready**: Health checks, structured logging
- 🔧 **Configurable**: Environment variables & CLI flags


<a id="quick-start"></a>
## 🚀 Quick Start

### Installation
```bash
# Install with go install (requires Go 1.23+)
go install github.com/johnwmail/nclip@latest

# Download pre-built binary
wget https://github.com/johnwmail/nclip/releases/latest/download/nclip_linux_amd64.tar.gz
tar -xzf nclip_linux_amd64.tar.gz
cd nclip_linux_amd64
# Run nclip from this directory to ensure static/ assets are found
./nclip

# Build from source
git clone https://github.com/johnwmail/nclip.git
cd nclip
go build -o nclip .

# Custom URL and TTL
export NCLIP_URL=https://demo.nclip.app
export NCLIP_TTL=24h
./nclip
```

### Client Usage Examples
```bash
# Start the service (automatically uses local filesystem in server mode)
./nclip

# Upload content via curl
echo "Hello World!" | curl -sL --data-binary @- http://localhost:8080
# Returns: http://localhost:8080/2F4D6

# Access content
curl -sL http://localhost:8080/2F4D6          # HTML view
curl -sL http://localhost:8080/raw/2F4D6      # Raw content

# Upload with base64 encoding (bypass WAF/security filters)
cat script.sh | base64 | curl -sL --data-binary @- http://localhost:8080/base64
# Server automatically decodes and stores original content

# Slug length: Slugs must be 3–32 characters. If out of range, default is 5.

# Web interface
open http://localhost:8080
```

For comprehensive client usage examples with curl, wget, PowerShell, HTTPie, and advanced features (custom TTL, slugs, base64, burn, etc.), see:

👉 **[Documents/CLIENTS.md](Documents/CLIENTS.md)** - Complete client usage guide

For detailed reference on custom HTTP headers (X-Base64, X-Burn, X-TTL, X-Slug, etc.), see:

👉 **[Documents/X-HEADERS.md](Documents/X-HEADERS.md)** - Header behavior, examples, and middleware/route precedence


<a id="deployment"></a>
## ☁️ Deployment

### Quick Start Options

| Method | Use Case | Setup Time | Scaling |
|--------|----------|------------|---------|
| **Docker** | Local development, small deployments | 2 minutes | Single instance |
| **Kubernetes** | Production, high availability | 10 minutes | Auto-scaling |
| **AWS Lambda** | Serverless, pay-per-use | 15 minutes | Automatic |

---

<a id="docker-deployment"></a>
## 🐳 Docker Deployment

### Quick Start (Recommended)
```bash
# Clone and run with Docker Compose
git clone https://github.com/johnwmail/nclip.git
cd nclip
docker-compose up -d
```

**Access:** http://localhost:8080

### Manual Docker Setup
```bash
# Pull and run the official image
docker run -d -p 8080:8080 --name nclip ghcr.io/johnwmail/nclip:latest
```

<a id="kubernetes-deployment"></a>
## ☸️ Kubernetes Deployment

### Quick Start
```bash
# Use the provided Kubernetes manifests
kubectl apply -f k8s/
```

📋 **[Kubernetes Guide](Documents/KUBERNETES.md)** - Complete deployment, scaling, and monitoring instructions

---

<a id="aws-lambda-deployment"></a>
## ☁️ AWS Lambda Deployment

### Overview
nclip automatically detects AWS Lambda environment and switches to S3 storage for serverless deployment.

### Prerequisites
1. **AWS Account** with appropriate permissions
2. **S3 Bucket** for paste storage
3. **IAM Role** with S3 permissions

### Quick Setup
```bash
# 1. Create S3 bucket
aws s3api create-bucket --bucket your-nclip-bucket --region us-east-1

# Build for Lambda
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o bootstrap .

# Create deployment package
zip lambda-function.zip bootstrap

# Create/update Lambda function
aws lambda create-function \
    --function-name your-nclip-function \
    --runtime provided.al2023 \
    --role arn:aws:iam::ACCOUNT:role/nclip-lambda-role \
    --handler bootstrap \
    --timeout 10 \
    --zip-file fileb://lambda-function.zip \
    --environment "Variables={NCLIP_S3_BUCKET=your-bucket,GIN_MODE=release}"
```

### IAM Permissions Required
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogGroup",
                "logs:CreateLogStream",
                "logs:PutLogEvents",
                "s3:GetObject",
                "s3:PutObject",
                "s3:DeleteObject",
                "s3:ListBucket"
            ],
            "Resource": "*"
        }
    ]
}
```

⚠️ **Buffer Size Note**: AWS Lambda has a 6MB total payload limit (including headers). See buffer size configuration details in the Lambda guide below.

📋 **[Lambda Guide](Documents/LAMBDA.md)** - Complete AWS Lambda deployment, monitoring, and troubleshooting

---


<a id="configuration"></a>
## ⚙️ Configuration

nclip supports configuration via environment variables and CLI flags. Environment variables take precedence over CLI flags.

### Environment Variables

| Variable | CLI Flag | Default | Description |
|----------|----------|---------|-------------|
| `NCLIP_PORT` | `--port` | `8080` | HTTP port to listen on |
| `NCLIP_URL` | `--url` | `""` | Base URL for paste links (auto-detected if empty) |
| `NCLIP_SLUG_LENGTH` | `--slug-length` | `5` | Length of generated slugs (3-32 characters) |
| `NCLIP_BUFFER_SIZE` | `--buffer-size` | `5242880` | Maximum upload size in bytes (5MB) |
| `NCLIP_TTL` | `--ttl` | `24h` | Default paste expiration time |
| `NCLIP_S3_BUCKET` | `--s3-bucket` | `""` | S3 bucket name for Lambda mode |
| `NCLIP_S3_PREFIX` | `--s3-prefix` | `""` | S3 key prefix for Lambda mode |
| `NCLIP_UPLOAD_AUTH` | `--upload-auth` | `false` | Require API key for upload endpoints |
| `NCLIP_API_KEYS` | `--api-keys` | `""` | Comma-separated API keys for upload authentication |
| `NCLIP_MAX_RENDER_SIZE` | `--max-render-size` | `262144` | Maximum size (bytes) to render inline in the HTML view; also used as preview length when content exceeds this size |

### API Key Authentication

Optionally require API keys for upload endpoints to prevent unauthorized usage. This is disabled by default.

**Enable Authentication:**
```bash
export NCLIP_UPLOAD_AUTH=true
export NCLIP_API_KEYS="secret-key-1,secret-key-2,secret-key-3"
./nclip
```

**Upload with API Key (curl):**
```bash
# Using Authorization: Bearer header
echo "protected content" | curl -sL --data-binary @- \
  -H "Authorization: Bearer secret-key-1" \
  http://localhost:8080

# Using X-Api-Key header
echo "protected content" | curl -sL --data-binary @- \
  -H "X-Api-Key: secret-key-1" \
  http://localhost:8080
```

**Upload with API Key (PowerShell):**
```powershell
# Using Authorization: Bearer header
$headers = @{ "Authorization" = "Bearer secret-key-1" }
"protected content" | Invoke-RestMethod -Uri http://localhost:8080 -Method Post -Headers $headers

# Using X-Api-Key header
$headers = @{ "X-Api-Key" = "secret-key-1" }
"protected content" | Invoke-RestMethod -Uri http://localhost:8080 -Method Post -Headers $headers
```

**Web UI:**
When API key authentication is enabled, the web UI includes an optional "API Key" input field. Simply paste your API key into this field before uploading content.

**Important Notes:**
- When `NCLIP_UPLOAD_AUTH=true`, all POST requests to `/` and `/burn/` require authentication
- GET requests (viewing/downloading pastes) do **not** require authentication
- When deployed behind CDNs (CloudFront/Cloudflare), ensure the distribution forwards `Authorization` or `X-Api-Key` headers to the origin
- Multiple API keys can be configured (comma-separated) for different users or applications

### Upload Auth (API Key) — additional guidance

When `NCLIP_UPLOAD_AUTH` is enabled, nclip enforces API key authentication for all upload endpoints (POST / and POST /burn/). This is intended to protect public-facing instances from abuse.

- Env var: `NCLIP_UPLOAD_AUTH=true` to enable
- Keys: `NCLIP_API_KEYS` should contain one or more comma-separated keys (for example: `key1,key2`).

Best practices and notes:

- Use secrets/parameters manager (Docker secrets, Kubernetes Secrets, or environment management) to avoid committing API keys into repo or images.
- For containers behind CDNs or proxies, make sure the proxy forwards either the `Authorization: Bearer <key>` header or the `X-Api-Key: <key>` header.
- Web UI: the UI will include an "API Key" input. Browsers will not send this automatically — paste or inject via browser extension if needed.
- Testing: for CI/integration tests you can set a single test key and pass it as `NCLIP_TEST_API_KEY` (used by integration scripts). This variable is optional and only used by the test harness.

Example (bash):

```
export NCLIP_UPLOAD_AUTH=true
export NCLIP_API_KEYS="secret-key-1,secret-key-2"
./nclip
```

### Examples

**Using Environment Variables:**
```bash
export NCLIP_PORT=3000
export NCLIP_URL=https://demo.nclip.app
export NCLIP_TTL=48h
./nclip
```

**Using CLI Flags:**
```bash
./nclip --port 3000 --url https://demo.nclip.app --ttl 48h
```

**Combined (Environment takes precedence):**
```bash
export NCLIP_PORT=3000
./nclip --url https://demo.nclip.app --ttl 48h
```

<a id="api-endpoints"></a>
## 📋 API Endpoints

### Core Endpoints
- `GET /` — Web UI (upload form, stats)
- `POST /` — Upload paste (returns URL, supports all headers)
- `POST /burn/` — Create burn-after-read paste (use `X-Burn` header)
- `POST /base64` — Upload base64-encoded content (use `X-Base64` header)
- `GET /{slug}` — HTML view of paste
- `GET /raw/{slug}` — Raw content download

**Supported Headers:** `X-TTL`, `X-Slug`, `X-Base64`, `X-Burn`, `X-Api-Key` / `Authorization`

### Metadata API
- `GET /api/v1/meta/{slug}` — JSON metadata (no content)
- `GET /json/{slug}` — Alias for `/api/v1/meta/{slug}` (shortcut)

### System Endpoints
- `GET /health` — Health check (200 OK)

### Paste Metadata (JSON)

Returned by `GET /api/v1/meta/{slug}` or `GET /json/{slug}`. Does **not** include the actual content.

```json
{
  "id": "string",                       // Unique paste ID
  "created_at": "2025-09-17T12:34:56Z", // ISO8601 timestamp
  "expires_at": "2025-09-18T12:34:56Z", // ISO8601 (null if no expiry)
  "size": 12345,                        // Size in bytes
  "content_type": "text/plain",         // MIME type
  "burn_after_read": true,              // true if burn-after-read
  "read_count": 0                       // Number of times read
}
```

<a id="development"></a>
## 🔧 Development

### Requirements

- **Go**: 1.23 or higher (minimum supported version)
- **Docker**: For container builds and testing

### Build Strategy

nclip follows a compatibility-first approach:

- **Minimum Go Version**: 1.23 (in `go.mod`) - Required by AWS SDK v2
- **Build/Release Go Version**: 1.25 (latest) - Uses newest optimizations and security features
- **CI Testing**: Tests against Go 1.23, 1.24, and 1.25

This means your code runs on Go 1.23+ systems while benefiting from the latest compiler optimizations in production builds.

### Local Development

```bash
# Clone and build
git clone https://github.com/johnwmail/nclip.git
cd nclip
go mod download
go build -o nclip .

# Run with local filesystem
./nclip
```

### Running Tests
```bash
# Format, vet, and test
go fmt ./... && go vet ./... && go test -v ./...

# Linting
golangci-lint run

# Run integration tests
go run main.go
bash scripts/integration-tests.sh
```

<a id="monitoring"></a>
## 📊 Monitoring

- **Health Check**: `GET /health` - Returns 200 OK with system status
- **Structured Logging**: JSON format with request tracing

<a id="links"></a>
## 🔗 Links

- **Documentation**: [Documents/](Documents/)
- **GitHub Registry**: `docker pull ghcr.io/johnwmail/nclip`
- **GitHub**: https://github.com/johnwmail/nclip
- **Issues**: https://github.com/johnwmail/nclip/issues

---

⭐ **Star this repository if you find it useful!**

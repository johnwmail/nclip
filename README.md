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
- [Features](#-features)
- [Quick Start](#-quick-start)
- [API Endpoints](#-api-endpoints)
- [Client Usage Examples](#-usage-examples)
- [Storage Architecture](#storage-architecture)
- [Configuration](#-configuration)
- [Deployment](#deployment)
  - [Docker](#-docker-deployment)
  - [Kubernetes](#kubernetes)
  - [AWS Lambda](#-aws-lambda-deployment)
- [Monitoring](#-monitoring)
- [Development](#-development)
- [Contributing](#-contributing)
- [License](#-license)
- [Links](#-links)

## Overview

nclip is a versatile clipboard app that accepts content via:
- **Web UI** - Browser interface at `http://localhost:8080`
- **Curl** - Modern web API: `echo "text" | curl -sL --data-binary @- http://localhost:8080`
- **File upload** - Upload (small) files via web UI or curl: `curl -sL --data-binary @/path/file http://localhost:8080`
- **Raw access** - Access raw content via `http://localhost:8080/raw/SLUG`
- **Burn after reading** - Content that self-destructs after being accessed once

## ✨ Features

🚀 **Dual Deployment**: Server mode (local or container) + AWS Lambda
🎯 **Unified Codebase**: Same code, logic, and UI for both environments
🗄️ **Multi-Storage Backend**: Filesystem for server mode, S3 for Lambda
🐳 **Container Ready**: Docker & Kubernetes deployment
⏰ **Auto-Expiration**: TTL support with configurable defaults
🛡️ **Production Ready**: Health checks, structured logging
🔧 **Configurable**: Environment variables & CLI flags

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

# Slug length: Slugs must be 3–32 characters. If out of range, default is 5.

# Web interface
open http://localhost:8080
```

For comprehensive client usage examples with curl, wget, PowerShell, HTTPie, and advanced features (custom TTL, slugs, etc.), see:

👉 **[docs/CLIENTS.md](docs/CLIENTS.md)** - Complete client usage guide


## 📋 API Endpoints

### Core Endpoints
- `GET /` — Web UI (upload form, stats)
- `POST /` — Upload paste (returns URL)
- `POST /burn/` — Create burn-after-read paste
- `GET /{slug}` — HTML view of paste
- `GET /raw/{slug}` — Raw content download

### Metadata API
- `GET /api/v1/meta/{slug}` — JSON metadata (no content)
- `GET /json/{slug}` — Alias for `/api/v1/meta/{slug}` (shortcut)

### System Endpoints
- `GET /health` — Health check (200 OK)

### Configuration
```bash
# Custom port and URL
### Environment Variables
All main configuration is via these environment variables (all have CLI flag equivalents):

# Environment variables
export NCLIP_URL=https://demo.nclip.app
export NCLIP_TTL=24h
./nclip
```

## � Deployment

nclip supports multiple deployment methods: Docker, Kubernetes, and AWS Lambda. Choose the deployment that best fits your needs.

### Quick Start Options

| Method | Use Case | Setup Time | Scaling |
|--------|----------|------------|---------|
| **Docker** | Local development, small deployments | 2 minutes | Single instance |
| **Kubernetes** | Production, high availability | 10 minutes | Auto-scaling |
| **AWS Lambda** | Serverless, pay-per-use | 15 minutes | Automatic |

---

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

## ☸️ Kubernetes Deployment

### Quick Start
```bash
# Use the provided Kubernetes manifests
kubectl apply -f k8s/
```

📋 **[Kubernetes Guide](docs/KUBERNETES.md)** - Complete deployment, scaling, and monitoring instructions

---

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
                "s3:HeadObject"
            ],
            "Resource": "*"
        }
    ]
}
```

📋 **[Lambda Guide](docs/LAMBDA.md)** - Complete AWS Lambda deployment, monitoring, and troubleshooting

---

## 🗄️ Storage Backends

| Deployment | Content Storage | Metadata Storage | TTL Support |
|------------|----------------|------------------|-------------|
| **Docker/K8s** | Filesystem | Filesystem | App logic |
| **AWS Lambda** | S3 | S3 | App logic |

**Storage selection is automatic** - no configuration needed. nclip detects the deployment environment and chooses the appropriate storage backend.

## 📊 Paste Metadata (JSON)

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

## �� Monitoring

- **Health Check**: `GET /health` - Returns 200 OK with system status
- **Structured Logging**: JSON format with request tracing

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

### Project Structure
```
/
├── main.go              # Unified entry point (server mode + Lambda)
├── main_test.go         # Integration tests
├── config/              # Configuration management
│   ├── config.go        # Configuration loading from env vars and CLI flags
│   └── config_test.go   # Configuration tests
├── handlers/            # HTTP request handlers
│   ├── paste.go         # Main paste upload/retrieval handler
│   ├── paste_test.go    # Paste handler tests
│   ├── meta.go          # Metadata API handler
│   ├── meta_test.go     # Metadata handler tests
│   ├── system.go        # System endpoints (health, etc.)
│   ├── system_test.go   # System handler tests
│   ├── webui.go         # Web UI handler
│   ├── webui_test.go    # Web UI tests
│   ├── retrieval/       # Paste retrieval handlers
│   └── upload/          # Paste upload handlers
├── internal/            # Private application code
│   └── services/        # Business logic services
│       └── paste_service.go # Paste business logic
├── models/              # Data models and structures
│   ├── paste.go         # Paste data model
│   └── paste_test.go    # Paste model tests
├── storage/             # Storage abstraction layer
│   ├── interface.go     # PasteStore interface definition
│   ├── interface_test.go # Interface tests
│   ├── filesystem.go    # Filesystem storage (server mode)
│   ├── filesystem_test.go # Filesystem storage tests
│   ├── s3.go            # S3 storage (Lambda mode)
│   ├── s3_test.go       # S3 storage tests
│   ├── s3util.go        # S3 utility functions
│   ├── s3util_test.go   # S3 utility tests
│   └── storage_test.go  # Storage integration tests
├── utils/               # Shared utilities
│   ├── debug.go         # Debug logging utilities
│   ├── debug_test.go    # Debug utility tests
│   ├── mime.go          # MIME type detection
│   ├── mime_test.go     # MIME detection tests
│   ├── slug.go          # Slug generation utilities
│   └── slug_test.go     # Slug generation tests
├── static/              # Static web assets
│   ├── index.html       # Main web UI
│   ├── favicon.ico      # Favicon
│   ├── style.css        # CSS styles
│   ├── script.js        # JavaScript functionality
│   └── view.html        # Paste view template
├── docs/                # Documentation
│   ├── CLIENTS.md       # Client usage examples
│   ├── CONTAINER_CLEANUP.md # Container management
│   ├── INTEGRATION-TESTS.md # Integration testing
│   ├── KUBERNETES.md    # Kubernetes deployment
│   └── LAMBDA.md        # AWS Lambda deployment
├── k8s/                 # Kubernetes manifests
│   ├── deployment.yaml  # Deployment configuration
│   ├── service.yaml     # Service configuration
│   ├── ingress.yaml     # Ingress configuration
│   ├── namespace.yaml   # Namespace definition
│   ├── kustomization.yaml # Kustomize configuration
│   └── pvc.yaml         # Persistent volume claim
├── scripts/             # Utility scripts
│   └── integration-test.sh # Integration test runner
├── .github/             # GitHub configuration
│   └── workflows/       # GitHub Actions workflows
├── Dockerfile           # Docker image definition
├── docker-compose.yml   # Docker Compose configuration
├── go.mod               # Go module definition
├── go.sum               # Go module checksums
├── .golangci.yml        # Go linting configuration
└── .gitignore           # Git ignore rules
```

## 🔗 Links

- **Documentation**: [docs/](docs/)
- **GitHub Registry**: `docker pull ghcr.io/johnwmail/nclip`
- **GitHub**: https://github.com/johnwmail/nclip
- **Issues**: https://github.com/johnwmail/nclip/issues

---

⭐ **Star this repository if you find it useful!**

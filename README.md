[![Test](https://github.com/johnwmail/nclip/workflows/Test/badge.svg)](https://github.com/johnwmail/nclip/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/johnwmail/nclip)](https://goreportcard.com/report/github.com/johnwmail/nclip)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub release](https://img.shields.io/github/release/johnwmail/nclip.svg)](https://github.com/johnwmail/nclip/releases)
[![Go Version](https://img.shields.io/badge/go-1.23+-blue.svg)](https://golang.org/)

# NCLIP

A modern, high-performance HTTP clipboard app written in Go with Gin framework.

## Storage Architecture

- **Lambda mode:** Content is stored in S3 as objects (`$slug`), with metadata in a JSON file (`$slug.json`).
- **Server mode:** Content is stored in the local filesystem as files (`$slug`), with metadata in a JSON file (`$slug.json`).
- **Metadata** includes expiry, burn-after-read, content type, and other small fields.
- This design keeps logic and code nearly identical between Lambda and server modes.

## Overview

nclip is a versatile clipboard app that accepts content via:
- **Web UI** - Browser interface at `http://localhost:8080`
- **Curl** - Modern web API: `echo "text" | curl --data-binary @- http://localhost:8080`
- **File upload** - Upload (small) files via web UI or curl: `curl --data-binary @/path/file http://localhost:8080`
- **Raw access** - Access raw content via `http://localhost:8080/raw/SLUG`
- **Burn after reading** - Content that self-destructs after being accessed once

## ✨ Features

🚀 **Dual Deployment**: Server mode (local or container) + AWS Lambda
🎯 **Unified Codebase**: Same code, logic, and UI for both environments
🗄️ **Multi-Storage Backend**: Filesystem for server mode, S3 for Lambda
🐳 **Server Ready**: Docker & Kubernetes deployment
- ⏰ **Auto-Expiration**: TTL support with configurable defaults
- 🛡️ **Production Ready**: Health checks, structured logging
- 🔧 **Configurable**: Environment variables & CLI flags

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

### Basic Usage
```bash
# Start the service (automatically uses local filesystem in server mode)
./nclip

# Upload content via curl
echo "Hello World!" | curl --data-binary @- http://localhost:8080
# Returns: http://localhost:8080/2F4D6

# Access content
curl http://localhost:8080/2F4D6          # HTML view
curl http://localhost:8080/raw/2F4D6      # Raw content

# Slug length: Slugs must be 3–32 characters. If out of range, default is 5.

# Web interface
open http://localhost:8080
```



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

*Access content via `/raw/{slug}` or `/{slug}`, not via metadata.*

## 📋 Usage Examples

### Command Line
```bash
# Upload text
echo "Secret message" | curl --data-binary @- http://localhost:8080

# Upload text file
curl --data-binary @myfile.txt http://localhost:8080

# Upload binary file
curl --data-binary @document.pdf http://localhost:8080

# Create burn-after-read paste
echo "Self-destruct message" | curl --data-binary @- http://localhost:8080/burn/

# Get metadata as JSON
curl http://localhost:8080/json/2F4D6
curl http://localhost:8080/api/v1/meta/2F4D6

### PowerShell / Windows
```powershell
# Post plain text
Invoke-WebRequest -Uri http://localhost:8080 -Method POST -Body "Hello from PowerShell!" -UseBasicParsing

# Post a text or binary file
Invoke-WebRequest -Uri http://localhost:8080 -Method POST -InFile "C:\path\to\file.txt" -UseBasicParsing

# Using Invoke-RestMethod (for text)
Invoke-RestMethod -Uri http://localhost:8080 -Method POST -Body "Hello from PowerShell!"

# Using Invoke-RestMethod (for retire)
Invoke-RestMethod -Uri http://localhost:8080/raw/2F4D6
```

## 🐚 Bash Aliases

You may find these bash aliases useful for working with nclip:

```bash
alias nclip="_nclip"
_nclip() {
  local _URL="https://demo.nclip.app"
  if [ -t 0 ]; then
    if [ $# -eq 1 ] && [ -f "$1" ]; then
      curl -sL --data-binary @"$1" "$_URL"
    else
      echo -en "$*" | curl -sL --data-binary @- "$_URL"
    fi
  else
    cat | curl -sL --data-binary @- "$_URL"
  fi
}
```

### Configuration
```bash
# Custom port and URL
./nclip --port 8080 --url https://demo.nclip.app

# Custom TTL and buffer size
./nclip --ttl 48h --buffer-size 5242880  # 5MB max

# Environment variables
export NCLIP_URL=https://demo.nclip.app
export NCLIP_TTL=24h
./nclip
```


## 🐳 Docker Deployment

### Quick Start with Docker Compose
```bash
# Clone and start with the included docker-compose.yml
git clone https://github.com/johnwmail/nclip.git
cd nclip
docker-compose up -d

# Or use the example below
```

### Docker Compose (with local filesystem)
```yaml
services:
  nclip:
    image: johnwmail/nclip:latest
    ports:
      - "8080:8080"
    environment:
      - NCLIP_URL=https://demo.nclip.app
    volumes:
      - ./data:/data  # Persist data to local ./data directory
```

### Production Docker Compose
```bash
# The repository includes a production-ready docker-compose.yml
# with health checks and volume mappings
docker-compose up -d
```

kubectl create deployment nclip --image=nclip
kubectl expose deployment nclip --port=8080 --type=LoadBalancer
### Kubernetes
```bash
# Deploy to Kubernetes with local filesystem (server mode)
kubectl apply -f k8s/nclip-filesystem.yaml

# Or build and deploy
docker build -t nclip .
kubectl create deployment nclip --image=nclip
kubectl expose deployment nclip --port=8080 --type=LoadBalancer
See [docs/KUBERNETES.md](docs/KUBERNETES.md) for detailed Kubernetes deployment instructions.
```

## ☁️ AWS Lambda Deployment

nclip automatically switches to S3 for storage when deployed as AWS Lambda (detected via `AWS_LAMBDA_FUNCTION_NAME`).

### Prerequisites
```bash
# Create S3 bucket
aws s3api create-bucket --bucket my-nclip-bucket --region us-east-1

# Enable versioning and lifecycle rules (optional)
aws s3api put-bucket-versioning --bucket my-nclip-bucket --versioning-configuration Status=Enabled
aws s3api put-bucket-lifecycle-configuration --bucket my-nclip-bucket --lifecycle-configuration file://lifecycle.json
```

### Deploy via GitHub Actions

When you push to the `deploy/lambda` branch, a GitHub Actions workflow automatically builds and deploys the Lambda function using the configuration in [`.github/workflows/lambda.yml`](.github/workflows/lambda.yml).
```bash
# Push to lambda deployment branch
git push origin deploy/lambda
```
- `GIN_MODE=release`

> **Note:** Ensure your Lambda function has appropriate AWS credentials and an IAM role with permissions for S3 access (e.g., `s3:GetObject`, `s3:PutObject`, `s3:DeleteObject` on the target bucket).

## 🗄️ Storage Backends

| Deployment         | Content Storage      | Metadata Storage      | TTL Support         |
|--------------------|---------------------|----------------------|---------------------|
| **Server mode**    | Filesystem (`$slug`)| Filesystem (`$slug.json`)| Handled by app logic |
| **AWS Lambda**     | S3 (`$slug`)        | S3 (`$slug.json`)| Handled by app logic |

Storage selection is automatic based on deployment environment - no configuration needed.

## ⚙️ Configuration

nclip supports configuration via environment variables and CLI flags.

### Environment Variables
```bash

# Server configuration
NCLIP_PORT=8080                    # HTTP port
NCLIP_URL=https://demo.nclip.app   # Base URL for paste links
NCLIP_SLUG_LENGTH=5                # Slug length (must be 3–32, default 5 if out of range)
NCLIP_BUFFER_SIZE=1048576          # Max upload size (1MB)
NCLIP_TTL=24h                      # Default paste expiration

# Storage configuration
NCLIP_S3_BUCKET=my-nclip-bucket         # S3 bucket for Lambda
```

### CLI Flags
All environment variables have corresponding CLI flags:
```bash
./nclip --port 8080 --url https://demo.nclip.app --ttl 48h
```

## 📊 Monitoring

- **Health Check**: `GET /health` - Returns 200 OK with system status
- **Structured Logging**: JSON format with request tracing

## 🔧 Development

### Running Tests
```bash
# Format, vet, and test
go fmt ./... && go vet ./... && go test -v ./...

# Linting
golangci-lint run

# Build for different platforms
GOOS=linux GOARCH=amd64 go build -o nclip-linux-amd64 .
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o bootstrap .  # Lambda
```

### Project Structure
```
/
├── main.go              # Unified entry point (server mode + Lambda)
├── config/              # Configuration management
├── storage/             # Storage interface & implementations
│   ├── interface.go     # PasteStore interface
│   ├── filesystem.go    # Filesystem (server mode) implementation
│   └── s3.go            # S3 (Lambda) implementation
├── handlers/            # HTTP request handlers
├── models/              # Data models
├── static/              # Web UI assets
└── utils/               # Utilities (slug generation, MIME detection)
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Go](https://golang.org/) and [Gin](https://gin-gonic.com/)
- Supports modern cloud-native deployments

## 🗂️ Container Registry Management

The repository includes automated cleanup of old container images to manage storage costs:

- **Automated Cleanup**: Monthly cleanup of container images older than 30 days
- **Manual Control**: Trigger cleanup with custom retention policies via GitHub Actions
- **Safe Deletion**: Always preserves `latest` tag and recent versions
- **Dry Run Mode**: Preview cleanup actions before execution

📋 **[Container Cleanup Guide](docs/CONTAINER_CLEANUP.md)** - Complete documentation for managing container images

## �️ Development

### Requirements

- **Go**: 1.23 or higher (minimum supported version)
- **Docker**: For container builds
  # ...existing code...

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

# Run tests
go test ./...

# Run with local filesystem
./nclip

# Run integration tests
make integration-tests
```

## �🔗 Links

- **Documentation**: [docs/](docs/)
- **GitHub Registry**: `docker pull ghcr.io/johnwmail/nclip`
- **GitHub**: https://github.com/johnwmail/nclip
- **Issues**: https://github.com/johnwmail/nclip/issues

---

⭐ **Star this repository if you find it useful!**

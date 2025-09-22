
[![Test](https://github.com/johnwmail/nclip/workflows/Test/badge.svg)](https://github.com/johnwmail/nclip/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/johnwmail/nclip)](https://goreportcard.com/report/github.com/johnwmail/nclip)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub release](https://img.shields.io/github/release/johnwmail/nclip.svg)](https://github.com/johnwmail/nclip/releases)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org/)

# NCLIP

A modern, high-performance HTTP clipboard service written in Go with Gin framework.

## Overview

nclip is a versatile clipboard service that accepts content via:
- **Web UI** - Browser interface at `http://localhost:8080`
- **Curl** - Modern web API: `echo "text" | curl --data-binary @- http://localhost:8080`
- **File upload** - Upload (small) files via web UI or curl: `curl --data-binary @/path/file http://localhost:8080`
- **Raw access** - Access raw content via `http://localhost:8080/raw/SLUG`
- **Burn after reading** - Content that self-destructs after being accessed once

## ✨ Features

- 🚀 **Dual Deployment**: Container/Kubernetes (MongoDB) + AWS Lambda (DynamoDB)
- 🎯 **Unified Codebase**: Same code, logic, and UI for both environments
- 🗄️ **Multi-Storage Backend**: MongoDB for containers, DynamoDB for serverless
- 🐳 **Container Ready**: Docker & Kubernetes deployment
- ⏰ **Auto-Expiration**: TTL support with configurable defaults
- 🛡️ **Production Ready**: Health checks, structured logging
- 🔧 **Configurable**: Environment variables & CLI flags

## 🚀 Quick Start

### Installation
```bash
# Download binary (replace with actual release)
wget https://github.com/johnwmail/nclip/releases/latest/download/nclip-linux-amd64
chmod +x nclip-linux-amd64
sudo mv nclip-linux-amd64 /usr/local/bin/nclip

# Or build from source
git clone https://github.com/johnwmail/nclip.git
cd nclip
go build -o nclip .
```

### Basic Usage
```bash
# Start the service (automatically uses MongoDB in container mode)
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

### Configuration
```bash
# Custom port and URL
./nclip --port 8080 --url https://paste.example.com

# Custom TTL and buffer size
./nclip --ttl 48h --buffer-size 5242880  # 5MB max

# Environment variables
export NCLIP_URL=https://paste.example.com
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

### Docker Compose (with MongoDB)
```yaml
services:
  nclip:
    image: johnwmail/nclip:latest
    ports:
      - "8080:8080"
    environment:
      - NCLIP_MONGO_URL=mongodb://mongo:27017
      - NCLIP_URL=https://paste.example.com
    depends_on:
      - mongo
  
  mongo:
    image: mongo:7
    volumes:
      - mongo_data:/data/db

volumes:
  mongo_data:
```

### Production Docker Compose
```bash
# The repository includes a production-ready docker-compose.yml
# with MongoDB initialization, TTL indexes, and health checks
docker-compose up -d
```

### Kubernetes
```bash
# Deploy to Kubernetes with MongoDB
kubectl apply -f k8s/nclip-mongodb.yaml

# Or build and deploy
docker build -t nclip .
kubectl create deployment nclip --image=nclip
kubectl expose deployment nclip --port=8080 --type=LoadBalancer
```

## ☁️ AWS Lambda Deployment

nclip automatically switches to DynamoDB when deployed as AWS Lambda (detected via `AWS_LAMBDA_FUNCTION_NAME`).

### Prerequisites
```bash
# Create DynamoDB table
aws dynamodb create-table \
    --table-name nclip-pastes \
    --attribute-definitions AttributeName=id,AttributeType=S \
    --key-schema AttributeName=id,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST \
    --stream-specification StreamEnabled=true,StreamViewType=NEW_AND_OLD_IMAGES
```

### Deploy via GitHub Actions

When you push to the `deploy/lambda` branch, a GitHub Actions workflow automatically builds and deploys the Lambda function using the configuration in [`.github/workflows/lambda.yml`](.github/workflows/lambda.yml).
```bash
# Push to lambda deployment branch
git push origin deploy/lambda
```
- `NCLIP_DYNAMO_TABLE=nclip-pastes`
- `GIN_MODE=release`

> **Note:** Ensure your Lambda function has appropriate AWS credentials and an IAM role with permissions for DynamoDB access (e.g., `dynamodb:GetItem`, `dynamodb:PutItem`, `dynamodb:UpdateItem`, `dynamodb:DeleteItem` on the target table).
- `NCLIP_DYNAMO_TABLE=nclip-pastes`
- `GIN_MODE=release`

## 🗄️ Storage Backends

| Deployment | Storage | Auto-Selected | TTL Support |
|------------|---------|---------------|-------------|
| **Container/K8s** | MongoDB | ✅ Automatic | Native TTL indexes |
| **AWS Lambda** | DynamoDB | ✅ Automatic | Native TTL attribute |

Storage selection is automatic based on deployment environment - no configuration needed!

## ⚙️ Configuration

nclip supports configuration via environment variables and CLI flags.

### Environment Variables
```bash

# Server configuration
NCLIP_PORT=8080                    # HTTP port
NCLIP_URL=https://paste.example.com # Base URL for paste links
NCLIP_SLUG_LENGTH=5                # Slug length (must be 3–32, default 5 if out of range)
NCLIP_BUFFER_SIZE=1048576          # Max upload size (1MB)
NCLIP_TTL=24h                      # Default paste expiration

# Feature toggles
NCLIP_ENABLE_WEBUI=true          # Web UI

# Storage configuration
NCLIP_MONGO_URL=mongodb://localhost:27017  # MongoDB connection
NCLIP_DYNAMO_TABLE=nclip-pastes             # DynamoDB table
```

### CLI Flags
All environment variables have corresponding CLI flags:
```bash
./nclip --port 8080 --url https://paste.example.com --ttl 48h
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
├── main.go              # Unified entry point (container + Lambda)
├── config/              # Configuration management
├── storage/             # Storage interface & implementations
│   ├── interface.go     # PasteStore interface
│   ├── mongodb.go       # MongoDB implementation
│   └── dynamodb.go      # DynamoDB implementation
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

## 🔗 Links

- **Documentation**: [docs/](docs/)
- **GitHub Registry**: `docker pull ghcr.io/johnwmail/nclip`
- **GitHub**: https://github.com/johnwmail/nclip
- **Issues**: https://github.com/johnwmail/nclip/issues

---

⭐ **Star this repository if you find it useful!**

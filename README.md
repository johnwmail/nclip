[![Test](https://github.com/johnwmail/nclip/workflows/Test/badge.svg)](https://github.com/johnwmail/nclip/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/johnwmail/nclip)](https://goreportcard.com/report/github.com/johnwmail/nclip)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub release](https://img.shields.io/github/release/johnwmail/nclip.svg)](https://github.com/johnwmail/nclip/releases)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org/)

# nclip

A modern, high-performance netcat-to-clipboard service written in Go, inspired by [fiche](https://github.com/solusipse/fiche).

## Overview

nclip is a dual-input clipboard service that accepts content via:
- **netcat (nc)** - Traditional command-line input: `echo "text" | nc localhost 9999`
- **HTTP/curl** - Modern web API: `echo "text" | curl -d @- http://localhost:8080`
- **HTTP/curl** - Web API with multilines support: `ps | curl --data-binary @- http://localhost:8080`
- **Web UI** - Browser interface at `http://localhost:8080`

## ✨ Features

- 🚀 **Dual Input Methods**: netcat + HTTP/curl + Web UI
- 🗄️ **Multi-Storage Backend**: filesystem, MongoDB, DynamoDB
- 🐳 **Container Ready**: Docker & Kubernetes deployment
- ⏰ **Auto-Expiration**: TTL support for all storage types
- 🛡️ **Production Ready**: Rate limiting, metrics, health checks
- 📈 **Scalable**: Horizontal pod autoscaling support
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
# Start the service
./nclip

# Using netcat (traditional)
echo "Hello World!" | nc localhost 9999

# Using curl (modern)
echo "Hello World!" | curl -d @- http://localhost:8080

# Web interface
open http://localhost:8080
```

## 📋 Usage Examples

### Command Line
```bash
# Share a file via netcat
cat myfile.txt | nc localhost 9999

# Share command output
ls -la | nc localhost 9999

# Share via HTTP with curl
echo "Secret message" | curl -d @- http://localhost:8080

# Upload a file via HTTP
curl -d @myfile.txt http://localhost:8080
```

### Configuration
```bash
# Custom ports and domain
./nclip -domain paste.example.com -tcp-port 9999 -http-port 8080

# Use MongoDB storage
./nclip -storage-type mongodb -mongodb-uri mongodb://localhost:27017

# Use DynamoDB storage
./nclip -storage-type dynamodb -dynamodb-table nclip-pastes

# Environment variables
export NCLIP_DOMAIN=paste.example.com
export NCLIP_STORAGE_TYPE=mongodb
./nclip
```

## 🐳 Docker Deployment

### Docker Compose
```bash
# Clone repository
git clone https://github.com/johnwmail/nclip.git
cd nclip

# Start with MongoDB
docker-compose up -d

# Access the service
echo "Hello Docker!" | nc localhost 9999
```

### Kubernetes
```bash
# Deploy to Kubernetes
kubectl apply -f k8s/nclip-mongodb.yaml

# Or with DynamoDB (for AWS)
kubectl apply -f k8s/nclip-dynamodb.yaml
```

## 🗄️ Storage Backends

| Backend | Status | Use Case | TTL Support |
|---------|--------|----------|-------------|
| **Filesystem** | ✅ Ready | Development, small deployments | Manual cleanup |
| **MongoDB** | ✅ Ready | Production, rich queries | Native TTL |
| **DynamoDB** | ✅ Ready | AWS serverless | Native TTL |

## ⚙️ Configuration

nclip supports configuration via environment variables and CLI flags. 

### Quick Configuration Examples
```bash
# Basic usage with custom URL
./nclip --url https://paste.example.com/clips/

# MongoDB storage
./nclip --storage-type mongodb --mongodb-uri mongodb://localhost:27017

# DynamoDB storage  
./nclip --storage-type dynamodb --dynamodb-table nclip-pastes

# Environment variables
export NCLIP_URL=https://nclip.app/paste/
export NCLIP_STORAGE_TYPE=mongodb
./nclip
```

📋 **[Complete Parameter Reference](docs/PARAMETER_REFERENCE.md)** - All environment variables, CLI flags, and configuration examples

## 🚀 Production Deployment

### AWS Lambda (Serverless)
```bash
# Use DynamoDB storage
NCLIP_STORAGE_TYPE=dynamodb
NCLIP_DYNAMODB_TABLE=nclip-pastes
```

### Docker Swarm / Kubernetes
```bash
# Use MongoDB for persistence
NCLIP_STORAGE_TYPE=mongodb
NCLIP_MONGODB_URI=mongodb://mongo-cluster:27017
```

## 📊 Monitoring

- **Health Check**: `GET /health`
- **Metrics**: `GET /metrics` (Prometheus format)
- **Stats**: `GET /stats` (JSON format)

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by [fiche](https://github.com/solusipse/fiche)
- Built with [Go](https://golang.org/)
- Powered by modern cloud-native technologies

## 🔗 Links

- **Documentation**: [docs/](docs/)
- **Docker Hub**: `docker pull johnwmail/nclip`
- **GitHub**: https://github.com/johnwmail/nclip
- **Issues**: https://github.com/johnwmail/nclip/issues

---

⭐ **Star this repository if you find it useful!**

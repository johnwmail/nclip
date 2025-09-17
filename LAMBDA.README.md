1. support http input only (no tcp input), use default http port 8080
2. support dynamodb storage only for lambda
3. support mongodb storage only for container/k8s deployment
4. support auto-expiration (TTL), 1 day by default (configurable)
5. since no tcp input support, the domain/host should be detected from http request header by default, but can be overridden by env var NCLIP_URL
6. support rate limiting, by default, 60 requests per minute for all clients, and 10 requests per minute for same I.P client (all configurable). The rate limit only for create new paste (read can be more )
7. support health check endpoint /health
8. support prometheus metrics endpoint /metrics (enabled by default, can be disabled by env var NCLIP_ENABLE_METRICS=false)
9. support webui at / (can be disabled by env var NCLIP_ENABLE_WEBUI=false)
10. the main webui / has a form to submit text/file to clipboard and show the resulting paste url, and simple stats information
11. support burn after reading, by URL /?burn=true or /burn
12. support raw text/file for raw output, by URL /raw or /?raw=true
13. the webui has raw and download buttons for easy access
13. the slug length is 5 by default (configurable by env var NCLIP_SLUG_LENGTH)
14. the paste/file max buffer size is 1mb by default (configurable by env var NCLIP_BUFFER_SIZE)
15. all configuration should be done by env vars, and one by one cli flags for each env var, for example, env var NCLIP_URL can be set by cli flag --url

15. the code must support go fmt, go vet, golangci-lint, and have unit tests with good coverage
16. single binary, no external dependencies except for mongodb/dynamodb
17. use same logic and code structure for lamdba/dynamodb and container/k8s/mongodb deployment, only the storage layer is different

18. After completed all above items. Please cleanup the project, remove obsolute/old/unused files and directories.


[![Test](https://github.com/johnwmail/nclip/workflows/Test/badge.svg)](https://github.com/johnwmail/nclip/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/johnwmail/nclip)](https://goreportcard.com/report/github.com/johnwmail/nclip)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub release](https://img.shields.io/github/release/johnwmail/nclip.svg)](https://github.com/johnwmail/nclip/releases)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org/)

# nclip

A modern, high-performance net-to-clipboard service written in Go, inspired by [fiche](https://github.com/solusipse/fiche).

## Overview

nclip is a http clipboard service that accepts content via:
- **HTTP/curl** - Modern web API: `echo "text" | curl -d @- http://localhost:8080`
- **HTTP/curl** - Web API with multilines support: `ps | curl --data-binary @- http://localhost:8080`
- **Web UI** - Browser interface at `http://localhost:8080`
- **File upload** - Upload files via web UI or curl: `curl -F 'file=@/path/to/file' http://localhost:8080`
- **Raw access** - Access raw content via `http://localhost:8080/raw/SLUG` or `http://localhost:8080/SLUG?raw=true`
- **Burn after reading** - Content that self-destructs after being accessed once via `http://localhost:8080/SLUG?burn=true` or `http://localhost:8080/burn/SLUG`
- **Prometheus metrics** - Metrics endpoint at `http://localhost:8080/metrics`

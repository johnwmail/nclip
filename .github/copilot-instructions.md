# Copilot Instructions for nclip

## Repository Overview

**nclip** is a modern netcat-to-clipboard service with multi-storage backend support. This is a Go-based project that provides network-based clipboard functionality, allowing users to send and receive clipboard data over the network with various storage backends.

### Current State
- **Status**: New repository with minimal setup
- **Language**: Go
- **Files Present**: Only `.gitignore` (Go-specific)
- **Expected Architecture**: CLI tool and/or service for network clipboard operations

## High-Level Repository Information

- **Project Type**: Go CLI application/service
- **Primary Language**: Go
- **Target Runtime**: Cross-platform (Linux, macOS, Windows)
- **Architecture**: Client-server or peer-to-peer clipboard service
- **Size**: Currently minimal (new repository)
- **Dependencies**: Will likely include networking, clipboard, and storage libraries

## Build and Development Instructions

### Prerequisites
- **Go Version**: Use Go 1.21 or later (check with `go version`)
- **Platform**: Cross-platform support expected

### Standard Go Project Commands

Since this is a Go project, follow these standard practices:

#### Bootstrap/Setup
```bash
# Initialize Go module (if not already done)
go mod init github.com/johnwmail/nclip

# Download dependencies
go mod download
go mod tidy
```

#### Build
```bash
# Build for current platform
go build -o nclip ./cmd/nclip

# Build for all platforms
GOOS=linux GOARCH=amd64 go build -o nclip-linux-amd64 ./cmd/nclip
GOOS=darwin GOARCH=amd64 go build -o nclip-darwin-amd64 ./cmd/nclip
GOOS=windows GOARCH=amd64 go build -o nclip-windows-amd64.exe ./cmd/nclip
```

#### Test
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...

# Run benchmarks
go test -bench=. ./...
```

#### Lint and Format
```bash
# Format code
go fmt ./...

# Run Go vet
go vet ./...

# Install and run golangci-lint (recommended)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
golangci-lint run
```

#### Run
```bash
# Run directly
go run ./cmd/nclip [args]

# Or after building
./nclip [args]
```

### Development Workflow
1. **Always run `go mod tidy` after adding dependencies**
2. **Format code with `go fmt ./...` before committing**
3. **Run `go vet ./...` to catch common issues**
4. **Run tests with `go test ./...` before submitting changes**
5. **Use `golangci-lint run` for comprehensive linting**

## Expected Project Layout

### Standard Go Project Structure
```
/
├── .github/                 # GitHub workflows and templates
├── cmd/                     # Main application entry points
│   └── nclip/              # Main CLI application
│       └── main.go
├── internal/               # Private application code
│   ├── config/            # Configuration handling
│   ├── clipboard/         # Clipboard operations
│   ├── network/           # Network communication
│   └── storage/           # Storage backend implementations
├── pkg/                   # Public library code (if any)
├── api/                   # API definitions (if applicable)
├── docs/                  # Documentation
├── scripts/               # Build and utility scripts
├── test/                  # Additional test files
├── .gitignore            # Git ignore file (present)
├── go.mod                # Go module file
├── go.sum                # Go module checksums
├── README.md             # Project documentation
├── LICENSE               # License file
└── Makefile              # Build automation (optional)
```

### Key Configuration Files
- **go.mod**: Go module definition and dependencies
- **go.sum**: Dependency checksums for security
- **.golangci.yml**: Linting configuration (if present)
- **Makefile**: Build automation (if present)
- **.github/workflows/**: CI/CD pipelines

## Architecture and Implementation Notes

### Expected Components
Based on the repository description, expect these architectural elements:

1. **Network Layer**: TCP/UDP server for receiving clipboard data
2. **Clipboard Interface**: Cross-platform clipboard operations
3. **Storage Backends**: Multiple storage options (memory, file, database, cloud)
4. **CLI Interface**: Command-line tool for client operations
5. **Configuration**: Support for various configuration formats (YAML, JSON, TOML)

### Key Dependencies (Likely)
- Clipboard operations: `github.com/atotto/clipboard` or similar
- CLI framework: `github.com/spf13/cobra` or `github.com/urfave/cli/v2`
- Configuration: `github.com/spf13/viper`
- Logging: `github.com/sirupsen/logrus` or `log/slog`
- Network: Standard library `net` package

### Development Practices
- Use Go modules for dependency management
- Follow Go naming conventions and package structure
- Implement proper error handling with wrapped errors
- Include comprehensive tests with table-driven test patterns
- Use interfaces for abstraction (especially for storage backends)
- Support graceful shutdown with context cancellation

## Validation and Testing

### Pre-commit Checklist
1. Run `go mod tidy` to clean up dependencies
2. Run `go fmt ./...` to format code
3. Run `go vet ./...` to check for issues
4. Run `go test ./...` to ensure tests pass
5. Run `golangci-lint run` for comprehensive linting
6. Test build with `go build ./...`

### CI/CD Expectations
Future GitHub Actions workflows should include:
- Go version matrix testing (1.21, 1.22, latest)
- Cross-platform builds (Linux, macOS, Windows)
- Linting with golangci-lint
- Security scanning with gosec
- Dependency vulnerability checks

### Manual Testing
For a clipboard service, manual testing should verify:
- Network connectivity and data transfer
- Clipboard integration on different platforms
- Storage backend functionality
- CLI command functionality
- Configuration file parsing

## Critical Instructions for Coding Agents

1. **Trust these instructions**: Only search for additional information if these instructions are incomplete or proven incorrect.

2. **Go-specific practices**: Always use `go mod tidy` after dependency changes, format with `go fmt`, and test with the race detector.

3. **Project is new**: Expect to create the basic project structure following Go conventions.

4. **Cross-platform consideration**: This tool should work on Linux, macOS, and Windows.

5. **Error handling**: Implement proper Go error handling patterns with error wrapping.

6. **Testing**: Write table-driven tests and include edge cases for network operations.

7. **Documentation**: Follow Go documentation conventions with package comments and exported function documentation.

## Time Expectations
- **go mod tidy**: 5-10 seconds
- **go build**: 10-30 seconds for initial build, 5-10 seconds for incremental
- **go test ./...**: Varies based on test complexity, typically 10-60 seconds
- **golangci-lint run**: 30-120 seconds depending on codebase size

Remember: This repository is currently minimal and will require initial project setup including `go.mod` creation, basic project structure, and implementation of the core netcat-to-clipboard functionality.
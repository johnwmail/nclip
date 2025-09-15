# Contributing to nclip

Thank you for your interest in contributing to nclip! This document provides guidelines and information about how to contribute to the project.

## 🚀 Quick Start

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/your-username/nclip.git
   cd nclip
   ```
3. **Install dependencies**:
   ```bash
   go mod download
   ```
4. **Run the tests** to ensure everything works:
   ```bash
   go test ./...
   ```

## 📋 Development Workflow

### Prerequisites

- Go 1.25+ installed
- Git for version control
- golangci-lint for code quality (optional but recommended)

### Making Changes

1. **Create a new branch** for your feature or bugfix:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the coding standards below

3. **Run quality checks**:
   ```bash
   # Format code
   go fmt ./...
   
   # Check for issues
   go vet ./...
   
   # Run tests
   go test ./...
   
   # Lint (if available)
   golangci-lint run
   ```

4. **Test your changes**:
   ```bash
   # Build and test locally
   go build -o nclip .
   ./nclip -log-level debug
   ```

5. **Commit your changes**:
   ```bash
   git add .
   git commit -m "feat: add awesome new feature"
   ```

6. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```

7. **Create a Pull Request** on GitHub

## 🎯 Coding Standards

### Go Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Use `go fmt` to format your code
- Write clear, descriptive variable and function names
- Add comments for exported functions and types
- Handle errors explicitly and appropriately

### Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation changes
- `style:` for formatting changes
- `refactor:` for code refactoring
- `test:` for adding tests
- `chore:` for maintenance tasks

Examples:
```
feat: add MongoDB storage backend support
fix: handle connection timeout in TCP server
docs: update deployment guide for Kubernetes
```

### Code Quality

All contributions must:

- ✅ Pass `go fmt ./...` (code formatting)
- ✅ Pass `go vet ./...` (static analysis)
- ✅ Pass `go test ./...` (all tests)
- ✅ Pass `golangci-lint run` (linting)
- ✅ Include tests for new functionality
- ✅ Update documentation as needed

## 🧪 Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run specific test
go test -v ./internal/storage
```

### Writing Tests

- Write unit tests for new functions and methods
- Use table-driven tests for multiple test cases
- Include edge cases and error conditions
- Aim for good test coverage (>80%)

Example test structure:
```go
func TestNewFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "test", "result", false},
        {"empty input", "", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := NewFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("NewFunction() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if result != tt.expected {
                t.Errorf("NewFunction() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

## 📚 Documentation

### Code Documentation

- Add comments for all exported functions, types, and constants
- Use Go doc comment conventions
- Include examples in comments when helpful

### Project Documentation

- Update README.md for user-facing changes
- Update deployment guides in `docs/` for operational changes
- Add new documentation files for significant features

## 🐛 Bug Reports

When reporting bugs, please include:

1. **Environment information**:
   - Go version
   - Operating system
   - nclip version/commit

2. **Steps to reproduce** the issue

3. **Expected vs actual behavior**

4. **Logs or error messages** (use `-log-level debug`)

5. **Configuration** used (anonymize sensitive data)

## 💡 Feature Requests

For new features:

1. **Check existing issues** to avoid duplicates
2. **Describe the use case** and problem being solved
3. **Propose a solution** or approach
4. **Consider backwards compatibility**
5. **Be willing to implement** or help with implementation

## 🔍 Code Review Process

All pull requests go through code review:

1. **Automated checks** must pass (CI/CD pipeline)
2. **Manual review** by maintainers
3. **Address feedback** promptly and respectfully
4. **Squash commits** if requested
5. **Rebase** on main branch before merge

### Review Criteria

- Code quality and style
- Test coverage and quality
- Documentation completeness
- Performance implications
- Security considerations
- Backwards compatibility

## 📞 Getting Help

- **GitHub Issues**: For bugs and feature requests
- **GitHub Discussions**: For questions and general discussion
- **Documentation**: Check existing docs in `docs/` folder

## 🎉 Recognition

Contributors will be recognized in:

- GitHub contributors list
- Release notes for significant contributions
- CONTRIBUTORS.md file (coming soon)

Thank you for contributing to nclip! 🚀

# Multi-stage build for smaller production image
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go modules files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=$VERSION -X main.buildTime=$BUILD_TIME -X main.gitCommit=$GIT_COMMIT" \
    -o nclip .

# Production stage
FROM alpine

# Install runtime dependencies
RUN apk add --no-cache ca-certificates curl

# Create non-root user
RUN addgroup -g 1001 -S nclip && \
    adduser -u 1001 -S nclip -G nclip

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/nclip .

# Create directories and set permissions
RUN mkdir -p pastes logs && \
    chown -R nclip:nclip /app

# Switch to non-root user
USER nclip

# Expose ports
EXPOSE 8080 8099

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Default configuration
ENV NCLIP_PORT=8080 \
    NCLIP_MONGO_URL=mongodb://localhost:27017

# Run the application
CMD ["./nclip"]

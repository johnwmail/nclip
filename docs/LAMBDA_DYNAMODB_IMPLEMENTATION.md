# Lambda DynamoDB Implementation

## Overview

This document describes the AWS Lambda implementation with DynamoDB storage backend for the nclip pastebin service. The Lambda deployment is optimized for serverless environments and **only supports DynamoDB storage** for simplicity and performance.

## Architecture

The unified implementation provides both serverless REST API and traditional server capabilities in a single binary. The binary **automatically detects the runtime environment** and switches between:

- **Lambda Mode**: When `AWS_LAMBDA_RUNTIME_API` or `_LAMBDA_SERVER_PORT` environment variables are present
- **Server Mode**: In all other cases, with full configuration support

### Environment Detection

- **Lambda Environment**: Uses DynamoDB handler exclusively
- **Container/Server Environment**: Supports filesystem, MongoDB, and DynamoDB storage backends

### Components

1. **Lambda Handler** (`internal/lambda/dynamodb/handler.go`)
   - Main HTTP request routing and processing
   - DynamoDB CRUD operations
   - Error handling and response formatting

2. **Unified Main** (`main.go`)
   - Single codebase with environment detection
   - Automatically switches between Lambda and server mode
   - Lambda mode: DynamoDB handler initialization
   - Server mode: Full configuration with multiple storage backends

3. **Tests** (`internal/lambda/dynamodb/handler_test.go`)
   - Unit tests for handler functions
   - Input validation testing
   - Error condition coverage

## Configuration

### Environment Variables

- `NCLIP_DYNAMODB_TABLE`: DynamoDB table name (default: `"nclip-pastes"`)
- `AWS_REGION`: AWS region for DynamoDB access

**Note**: Storage type is hardcoded to DynamoDB for Lambda deployments.

### DynamoDB Table Schema

```
Table Name: nclip-pastes
Primary Key: id (String)

Attributes:
- id (String) - Paste unique identifier
- content (String) - Paste content
- content_type (String) - MIME type
- created_at (Number) - Unix timestamp
- expires_at (Number) - Unix timestamp
- client_ip (String) - Client IP address
- size (Number) - Content size in bytes
- metadata (String) - JSON-encoded metadata
```

## API Endpoints

### POST /paste
Creates a new paste

**Request Body:**
```json
{
  "id": "unique-id",
  "content": "paste content",
  "content_type": "text/plain",
  "client_ip": "127.0.0.1",
  "size": 12,
  "metadata": {"key": "value"}
}
```

**Response:**
- `201 Created`: Paste created successfully
- `400 Bad Request`: Invalid request or missing paste ID
- `500 Internal Server Error`: DynamoDB operation failed

### GET /paste/{id}
Retrieves a paste by ID

**Response:**
- `200 OK`: Paste found and returned
- `400 Bad Request`: Missing paste ID
- `404 Not Found`: Paste not found
- `410 Gone`: Paste has expired
- `500 Internal Server Error`: DynamoDB operation failed

## Features

### Input Validation
- Validates JSON request format
- Requires paste ID for creation
- Validates path parameters for retrieval

### Error Handling
- Proper HTTP status codes
- Detailed error messages
- Graceful handling of DynamoDB errors

### Expiration Support
- Automatic expiration checking on retrieval
- Default 30-day expiration for new pastes
- Returns 410 Gone for expired pastes

### Content Type Headers
- Proper Content-Type headers in responses
- JSON responses for successful operations
- Plain text for error messages

## Deployment

### Using GitHub Actions

The project includes a deployment workflow (`.github/workflows/deploy-lambda.yml`) that:

1. Builds the Lambda binary for Linux
2. Creates a deployment package
3. Updates the Lambda function
4. Configures environment variables

### Manual Deployment

```bash
## Build and Deployment

### Unified Binary Approach

Both server and Lambda deployments use the same binary with automatic environment detection:

```bash
# Build for server deployment
go build -o nclip .

# Build for Lambda deployment (with CGO disabled)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bootstrap .
```

The binary automatically detects the runtime environment:
- **Lambda**: When AWS runtime environment variables are present
- **Server**: In all other cases

### Container Deployment

```bash
# The binary will run in server mode
./nclip --storage-backend mongodb --mongodb-uri "mongodb://localhost:27017"
```

### Lambda Deployment

```bash
# The binary will automatically detect Lambda environment and use DynamoDB
zip deployment.zip bootstrap
aws lambda update-function-code --function-name nclip --zip-file fileb://deployment.zip
```
```

## Testing

### Unit Tests

Run the handler tests:
```bash
go test ./internal/lambda/dynamodb/
```

### Environment Detection Testing

Test the unified binary behavior:
```bash
# Test server mode (default)
./nclip --help

# Test Lambda mode detection
AWS_LAMBDA_RUNTIME_API=test timeout 2s ./nclip
```

### Integration Testing

For full integration testing, deploy to a test environment with:
- DynamoDB table with proper permissions
- Lambda function with DynamoDB access
- API Gateway integration

## Implementation Notes

### Unified Main Function

The single `main.go` contains unified code that:

1. **Detects Runtime Environment**:
   ```go
   isLambda := os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" || os.Getenv("_LAMBDA_SERVER_PORT") != ""
   ```

2. **Routes to Appropriate Handler**:
   - Lambda mode: Simplified DynamoDB-only handler
   - Server mode: Full configuration support with multiple storage backends

3. **Maintains Code Consistency**: Single implementation eliminates duplication and reduces maintenance overhead

### Storage Backend Selection

- **Lambda Mode**: Exclusively uses DynamoDB (environment variables required)
- **Server Mode**: Supports filesystem, MongoDB, and DynamoDB based on configuration

### Error Handling

The unified approach provides consistent error handling across both deployment modes while respecting the constraints of each environment.

## Security Considerations

1. **IAM Permissions**: Lambda function needs DynamoDB read/write permissions
2. **Input Validation**: All inputs are validated before processing
3. **Error Handling**: Sensitive error details are not exposed to clients
4. **Expiration**: Automatic cleanup through TTL prevents data accumulation

## Performance

- DynamoDB provides consistent low-latency access
- Lambda cold starts are minimized through efficient initialization
- Proper error handling prevents cascade failures
- Efficient JSON serialization for metadata storage

## Monitoring

Recommended CloudWatch metrics to monitor:
- Lambda invocation count and duration
- DynamoDB read/write capacity utilization
- Error rates and types
- Response time percentiles

## Cost Optimization

- Use DynamoDB on-demand pricing for variable workloads
- Implement TTL to automatically delete expired pastes
- Monitor Lambda memory usage for right-sizing
- Consider reserved capacity for consistent high-volume usage

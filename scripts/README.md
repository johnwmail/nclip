# Integration Tests

This directory contains integration tests for the nclip service.

## Files

### `integration-test.sh`
Comprehensive integration test script that validates all nclip API endpoints against a real MongoDB backend.

**Test Coverage:**
- Health endpoint verification
- Paste creation via POST requests
- Paste retrieval via GET requests  
- Raw paste access without formatting
- Metadata API endpoints (both `/api/v1/meta/:slug` and `/json/:slug`)
- Burn-after-read functionality
- 404 error handling for non-existent pastes
- Metrics endpoint accessibility

**Usage:**
```bash
# Set target URL (defaults to http://localhost:8080)
export NCLIP_URL=http://localhost:8080

# Run tests
bash scripts/integration-test.sh
```

**Environment Variables:**
- `NCLIP_URL`: Target nclip service URL (default: http://localhost:8080)
- `TEST_TIMEOUT`: Test timeout in seconds (default: 30)
- `RETRY_DELAY`: Delay between retries in seconds (default: 2)
- `MAX_RETRIES`: Maximum number of retries for readiness check (default: 15)

### `mongo-init-test.js`
MongoDB initialization script for test environments. Creates necessary indexes for proper paste management and expiration.

**Features:**
- TTL index for automatic paste expiration
- Unique index on paste ID field
- Created timestamp index for queries

## GitHub Actions Integration

The integration tests are automatically run in the GitHub Actions workflow (`test.yml`) with:

- **MongoDB Service**: Real MongoDB 7.0 instance with health checks
- **Service Dependencies**: Tests run after unit tests and linting pass
- **Conditional Execution**: Only runs on main branch pushes and pull requests
- **Artifact Collection**: Failed tests upload debugging artifacts
- **Proper Cleanup**: Server processes are gracefully terminated

## Local Testing

To run integration tests locally:

1. Start MongoDB:
   ```bash
   docker run --name nclip-mongo -p 27017:27017 -d mongo:7.0
   ```

2. Initialize MongoDB:
   ```bash
   mongosh nclip < scripts/mongo-init-test.js
   ```

3. Build and start nclip:
   ```bash
   go build -o nclip .
   NCLIP_MONGO_URL=mongodb://localhost:27017 ./nclip &
   ```

4. Run tests:
   ```bash
   bash scripts/integration-test.sh
   ```

5. Cleanup:
   ```bash
   pkill nclip
   docker stop nclip-mongo && docker rm nclip-mongo
   ```

## Test Output

The integration tests provide colored output with clear success/failure indicators:
- 🔵 **[INFO]** - General information and progress
- 🟡 **[WARN]** - Warnings (non-critical issues)
- 🔴 **[ERROR]** - Test failures and errors
- 🟢 **[SUCCESS]** - Successful test completions

All tests must pass for the integration test suite to succeed.
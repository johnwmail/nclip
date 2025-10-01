# Integration Tests

This directory contains integration tests for the nclip service.

## Files

### `integration-test.sh`
Comprehensive integration test script that validates all nclip API endpoints against S3 and filesystem backends.

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

### MongoDB support removed
All MongoDB initialization and features have been removed. NCLIP now uses S3 and filesystem backends only.

## GitHub Actions Integration

The integration tests are automatically run in the GitHub Actions workflow (`test.yml`) with:

## GitHub Actions Integration
- **Service Dependencies**: Tests run after unit tests and linting pass
- **Conditional Execution**: Runs on main/dev branch pushes and pull requests
- **Artifact Collection**: Failed tests upload debugging artifacts
- **Proper Cleanup**: Server processes are gracefully terminated

## Local Testing

To run integration tests locally:

1. Build and start nclip:
   ```bash
   go build -o nclip .
   ```

2. Run tests:
   ```bash
   bash scripts/integration-test.sh
   ```

3. Cleanup:
   ```bash
   pkill nclip
   ```

## Test Output

The integration tests provide colored output with clear success/failure indicators:
- 🔵 **[INFO]** - General information and progress
- 🟡 **[WARN]** - Warnings (non-critical issues)
- 🔴 **[ERROR]** - Test failures and errors
- 🟢 **[SUCCESS]** - Successful test completions

All tests must pass for the integration test suite to succeed.
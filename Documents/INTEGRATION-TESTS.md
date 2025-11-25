# Integration Tests



This document describes the integration test suite for the nclip service.



## Overview## Files



The integration test suite validates all nclip API endpoints, features, and edge cases against S3 and filesystem backends. The tests are orchestrated by a unified script that runs individual test modules in sequence.### `integration-test.sh`

Comprehensive integration test script that validates all nclip API endpoints against S3 and filesystem backends.

## Main Test Runner

**Test Coverage:**

### `scripts/integration-test.sh`- Health endpoint verification

- Paste creation via POST requests

Unified integration test runner that orchestrates all test modules and provides comprehensive validation of the nclip service.- Paste retrieval via GET requests  

- Raw paste access without formatting

**Test Coverage:**- Metadata API endpoints (both `/api/v1/meta/:slug` and `/json/:slug`)

- Health endpoint verification (`test_health.sh`)- Burn-after-read functionality

- Paste creation and retrieval (`test_paste.sh`)- 404 error handling for non-existent pastes

- Burn-after-read functionality (`test_burn.sh`)- Metrics endpoint accessibility

- File extension handling for text and binary files (`test_extensions_*.sh`)

- Raw paste access without formatting**Usage:**

- Metadata API endpoints (both `/api/v1/meta/:slug` and `/json/:slug`)```bash

- 404 error handling for non-existent pastes (`test_not_found.sh`)# Set target URL (defaults to http://localhost:8080)

- TTL expiration logic (`test_xttl.sh`, `test_expired.sh`)export NCLIP_URL=http://localhost:8080

- Slug collision detection (`test_slug_collision.sh`)

- Buffer size limits (`test_buffer.sh`)# Run tests

- Preview mode for large files (`test_preview.sh`)bash scripts/integration-test.sh

- Size mismatch detection (`test_size_mismatch.sh`)```



**Usage:****Environment Variables:**

```bash- `NCLIP_URL`: Target nclip service URL (default: http://localhost:8080)

# Set target URL (defaults to http://localhost:8080)- `TEST_TIMEOUT`: Test timeout in seconds (default: 30)

export NCLIP_URL=http://localhost:8080- `RETRY_DELAY`: Delay between retries in seconds (default: 2)

- `MAX_RETRIES`: Maximum number of retries for readiness check (default: 15)

# Run all integration tests

bash scripts/integration-test.sh### MongoDB support removed

```All MongoDB initialization and features have been removed. NCLIP now uses S3 and filesystem backends only.



**Environment Variables:**## GitHub Actions Integration

- `NCLIP_URL`: Target nclip service URL (default: http://localhost:8080)

- `NCLIP_DATA_DIR`: Data directory path (default: ./data)The integration tests are automatically run in the GitHub Actions workflow (`test.yml`) with:

- `NCLIP_UPLOAD_AUTH`: Enable/disable upload authentication (true/false)

- `NCLIP_TEST_API_KEY`: API key for authenticated uploads (when UPLOAD_AUTH=true)## GitHub Actions Integration

- `NCLIP_TEST_API_KEY_BEARER`: Bearer token for authenticated uploads (alternative)- **Service Dependencies**: Tests run after unit tests and linting pass

- `TEST_TIMEOUT`: Test timeout in seconds (default: 30)- **Conditional Execution**: Runs on main/dev branch pushes and pull requests

- `RETRY_DELAY`: Delay between retries in seconds (default: 2)- **Artifact Collection**: Failed tests upload debugging artifacts

- `MAX_RETRIES`: Maximum number of retries for readiness check (default: 15)- **Proper Cleanup**: Server processes are gracefully terminated



## Test Modules## Local Testing



Individual test scripts are located in `scripts/integration/` and can be run independently for debugging:To run integration tests locally:



| Script | Description |1. Build and start nclip:

|--------|-------------|   ```bash

| `test_health.sh` | Health endpoint checks |   go build -o nclip .

| `test_paste.sh` | Basic paste create/retrieve operations |   ```

| `test_burn.sh` | Burn-after-read validation |

| `test_extensions_text.sh` | Text file extension handling |2. Run tests:

| `test_extensions_binary.sh` | Binary file extension handling |   ```bash

| `test_not_found.sh` | 404 error responses |   bash scripts/integration-test.sh

| `test_xttl.sh` | TTL header validation |   ```

| `test_slug_collision.sh` | Slug uniqueness enforcement |

| `test_expired.sh` | Expiration logic |3. Cleanup:

| `test_buffer.sh` | Buffer size limit enforcement |   ```bash

| `test_preview.sh` | Large file preview mode |   pkill nclip

| `test_size_mismatch.sh` | Size mismatch detection |   ```



### Shared Library## Test Output



`scripts/integration/lib.sh` provides common functions and utilities:The integration tests provide colored output with clear success/failure indicators:

- Colored logging (INFO, WARN, ERROR, SUCCESS)- 🔵 **[INFO]** - General information and progress

- HTTP request helpers with automatic authentication- 🟡 **[WARN]** - Warnings (non-critical issues)

- Test cleanup and artifact management- 🔴 **[ERROR]** - Test failures and errors

- Configuration defaults and environment setup- 🟢 **[SUCCESS]** - Successful test completions

- Retry logic for flaky network operations

All tests must pass for the integration test suite to succeed.
## GitHub Actions Integration

The integration tests run automatically in CI via `.github/workflows/integration-test.yml`:

**Features:**
- **Multi-version Testing**: Tests against Go 1.24 and 1.25
- **Auth Matrix**: Tests with both `UPLOAD_AUTH=true` and `UPLOAD_AUTH=false`
- **Service Dependencies**: Tests run after unit tests and linting pass
- **Conditional Execution**: Runs on main/dev branch pushes and pull requests
- **Artifact Collection**: Failed tests upload debugging artifacts (logs, binaries)
- **Proper Cleanup**: Server processes are gracefully terminated after tests

**Workflow Steps:**
1. Checkout code and setup Go environment
2. Build nclip binary with version injection
3. Start nclip server (filesystem backend)
4. Wait for health endpoint readiness
5. Run unified integration test suite
6. Stop server and cleanup
7. Upload artifacts on failure

## Local Testing

To run integration tests locally:

1. **Build and start nclip:**
   ```bash
   go build -o nclip .
   ./nclip &
   ```

2. **Run tests:**
   ```bash
   bash scripts/integration-test.sh
   ```

3. **Cleanup:**
   ```bash
   pkill nclip
   rm -rf ./data
   ```

### Testing with Authentication

To test upload authentication locally:

```bash
# Generate a test API key
export NCLIP_API_KEYS="test-key-12345"
export NCLIP_UPLOAD_AUTH=true
export NCLIP_TEST_API_KEY="test-key-12345"

# Start server with auth
./nclip &

# Run tests
bash scripts/integration-test.sh
```

## Test Output

The integration tests provide colored output with clear success/failure indicators:

- 🔵 **[INFO]** - General information and progress
- 🟡 **[WARN]** - Warnings (non-critical issues)
- 🔴 **[ERROR]** - Test failures and errors
- 🟢 **[SUCCESS]** - Successful test completions

**Example output:**
```
[INFO] Integration Test Environment:
[INFO]   NCLIP_URL: http://localhost:8080
[INFO]   NCLIP_DATA_DIR: ./data
[INFO]   Upload Auth: false

[INFO] Running test_health.sh...
[SUCCESS] test_health.sh PASSED

[INFO] Running test_paste.sh...
[SUCCESS] test_paste.sh PASSED

...

========================================
[SUCCESS] All integration tests PASSED (12/12)
```

## Test Cleanup

**CRITICAL**: All tests automatically clean up artifacts they create:
- `TRASH_RECORD_FILE="/tmp/nclip_integration_slugs.txt"` tracks created slugs
- Cleanup function removes only recorded slugs or recently modified files (`-mmin -60`)
- Never uses broad cleanup like `rm -rf ./data/*` - protects unrelated data
- Cleanup runs automatically on EXIT via trap handlers

This ensures reproducible, reliable tests and prevents leftover artifacts from affecting subsequent runs.

## Storage Backends

The integration tests validate both storage backends:
- **Filesystem**: Local disk storage (default for local testing)
- **S3**: AWS S3 storage (used in Lambda/production deployments)

MongoDB support has been removed. NCLIP now exclusively uses filesystem and S3 backends.

## Troubleshooting

### Tests fail with "connection refused"
- Ensure nclip server is running: `curl http://localhost:8080/health`
- Check if port 8080 is already in use: `lsof -i :8080`

### Tests fail with 401 Unauthorized
- Check if `NCLIP_UPLOAD_AUTH=true` is set
- Verify `NCLIP_TEST_API_KEY` matches `NCLIP_API_KEYS`

### Tests leave behind data files
- Tests should auto-cleanup via EXIT trap
- Manual cleanup: `rm -rf ./data /tmp/nclip_test_* /tmp/nclip_integration_slugs.txt`

### Individual test debugging
Run a specific test module directly:
```bash
source scripts/integration/lib.sh
bash scripts/integration/test_paste.sh
```

## Contributing

When adding new integration tests:

1. Create a new test script in `scripts/integration/test_*.sh`
2. Source `lib.sh` and use provided helper functions
3. Add the test to the `SCRIPTS` array in `scripts/integration-test.sh`
4. Ensure proper cleanup using `record_slug()` for created pastes
5. Use colored logging functions for clear output
6. Test both with and without upload authentication
7. Update this documentation with the new test coverage

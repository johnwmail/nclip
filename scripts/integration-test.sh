#!/bin/bash
set -euo pipefail

# Unified Integration Test Runner for nclip
# This script runs all integration tests against a running nclip instance
# and validates all API endpoints, features, and edge cases.

# Resolve directory paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INTEGRATION_DIR="$SCRIPT_DIR/integration"

# Source shared library functions
source "$INTEGRATION_DIR/lib.sh"

# Enable cleanup on exit
enable_cleanup_trap

# List of all integration test scripts to run
SCRIPTS=(
    "$INTEGRATION_DIR/test_health.sh"
    "$INTEGRATION_DIR/test_paste.sh"
    "$INTEGRATION_DIR/test_burn.sh"
    "$INTEGRATION_DIR/test_extensions_text.sh"
    "$INTEGRATION_DIR/test_extensions_binary.sh"
    "$INTEGRATION_DIR/test_not_found.sh"
    "$INTEGRATION_DIR/test_xttl.sh"
    "$INTEGRATION_DIR/test_slug_collision.sh"
    "$INTEGRATION_DIR/test_expired.sh"
    "$INTEGRATION_DIR/test_buffer.sh"
    "$INTEGRATION_DIR/test_preview.sh"
    "$INTEGRATION_DIR/test_size_mismatch.sh"
)

# Print test environment configuration
log "Integration Test Environment:"
log "  NCLIP_URL: ${NCLIP_URL}"
log "  NCLIP_DATA_DIR: ${NCLIP_DATA_DIR}"
log "  Upload Auth: ${NCLIP_UPLOAD_AUTH:-false}"
if [[ -n "${NCLIP_TEST_API_KEY:-}" ]]; then
    log "  Using API Key authentication"
elif [[ -n "${NCLIP_TEST_API_KEY_BEARER:-}" ]]; then
    log "  Using Bearer token authentication"
fi
echo

# Run all test scripts
failed=0
for script in "${SCRIPTS[@]}"; do
    echo
    log "Running $(basename "$script")..."
    if ! bash "$script"; then
        warn "$(basename "$script") FAILED"
        failed=$((failed+1))
    else
        success "$(basename "$script") PASSED"
    fi
done

echo
echo "========================================"
if [[ $failed -eq 0 ]]; then
    success "All integration tests PASSED (${#SCRIPTS[@]}/${#SCRIPTS[@]})"
    exit 0
else
    error "Integration tests FAILED: $failed/${#SCRIPTS[@]} failed"
    exit 1
fi
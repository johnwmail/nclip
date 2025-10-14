#!/bin/bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

test_health() {
    log "Testing health endpoint..."
    # Wait for the server to become ready (helper in lib.sh)
    if ! wait_for_nclip; then
        error "nclip did not become ready"
        return 1
    fi
    local response
    response=$(cget_f "$NCLIP_URL/health")
    if [[ "$response" == *"healthy"* ]] || [[ "$response" == *"ok"* ]] || [[ "$response" == *"OK"* ]]; then
        success "Health check passed: $response"
        return 0
    else
        error "Health check failed: $response"
        return 1
    fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    test_health
fi

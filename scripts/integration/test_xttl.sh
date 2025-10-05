#!/bin/bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

test_x_ttl_valid() {
    log "Testing valid X-TTL header (2h)..."
    local test_content="Valid X-TTL test $(date)"
    local ttl="2h"
    local paste_url
    paste_url=$(cpost -X POST "$NCLIP_URL/" -d "$test_content" -H "X-TTL: $ttl") || true
    if [[ -n "$paste_url" && "$paste_url" == http* ]]; then record_slug "$(basename "$paste_url")"; success "Valid X-TTL test passed: $paste_url"; return 0; else error "Valid X-TTL test failed. Response: $paste_url"; return 1; fi
}

test_x_ttl_invalid() {
    log "Testing invalid X-TTL header (<1h, >7d, and non-time string)..."
    local test_content="Invalid X-TTL test $(date)"
    local status
    status=$(cpost -s -o /dev/null -w "%{http_code}" -X POST "$NCLIP_URL/" -d "$test_content" -H "X-TTL: 30m")
    if [[ "$status" == "400" ]]; then success "Invalid X-TTL test (30m) passed: server rejected short TTL"; else error "Invalid X-TTL test (30m) failed: expected 400, got $status"; return 1; fi
    status=$(cpost -s -o /dev/null -w "%{http_code}" -X POST "$NCLIP_URL/" -d "$test_content" -H "X-TTL: 8d")
    if [[ "$status" == "400" ]]; then success "Invalid X-TTL test (8d) passed: server rejected long TTL"; else error "Invalid X-TTL test (8d) failed: expected 400, got $status"; return 1; fi
    status=$(cpost -s -o /dev/null -w "%{http_code}" -X POST "$NCLIP_URL/" -d "$test_content" -H "X-TTL: Time_invalid")
    if [[ "$status" == "400" ]]; then success "Invalid X-TTL test (Time_invalid) passed: server rejected non-time string"; return 0; else error "Invalid X-TTL test (Time_invalid) failed: expected 400, got $status"; return 1; fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cleanup_temp_files
    wait_for_nclip
    test_x_ttl_valid
    test_x_ttl_invalid
    cleanup_temp_files
fi

#!/bin/bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

test_expired_paste() {
    log "Testing expired paste behavior..."
    local test_content="Expired paste test $(date)"
    local ttl="1h"
    local paste_url
    paste_url=$(cpost -X POST "$NCLIP_URL/" -d "$test_content" -H "X-TTL: $ttl") || true
    if [[ -z "$paste_url" || ! "$paste_url" =~ http ]]; then error "Failed to create paste with TTL. Response: $paste_url"; return 1; fi
    log "Paste created with TTL: $paste_url"
    log "Skipping expiry check: backend does not support DELETE and TTL <1h is invalid."
    success "Expired paste test (creation only) passed. Manual expiry not tested."
    return 0
}

test_expired_paste_manual() {
    log "Testing expired paste behavior with manual file injection..."
    local slug="EXPIRED1"
    local now
    now=$(date -u +%Y-%m-%dT%H:%M:%S.%NZ)
    local expired
    expired=$(date -u -d "-1 day" +%Y-%m-%dT%H:%M:%S.%NZ)
    local dir="./data"
    mkdir -p "$dir"
    local file="$dir/${slug}.json"
    cat > "$file" <<EOF
{
  "id": "$slug",
  "created_at": "$now",
  "expires_at": "$expired",
  "size": 42,
  "content_type": "text/plain; charset=utf-8",
  "burn_after_read": false,
  "read_count": 0
}
EOF
    log "Injected expired paste file: $file"
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" "$NCLIP_URL/$slug")
    if [[ "$status" == "404" || "$status" == "400" ]]; then success "Manual expired paste test passed: server returned $status for expired paste"; rm -f "$file"; return 0; else error "Manual expired paste test failed: expected 404 or 400, got $status"; rm -f "$file"; return 1; fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cleanup_temp_files
    wait_for_nclip
    test_expired_paste
    test_expired_paste_manual
    cleanup_temp_files
fi

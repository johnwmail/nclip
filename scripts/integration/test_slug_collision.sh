#!/bin/bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

test_slug_collision() {
    log "Testing slug collision prevention..."
    local charset="ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
    local slug=""
    for ((i=0; i<5; i++)); do
        idx=$(od -An -N2 -tu2 < /dev/urandom | awk '{print $1 % 32}')
        slug+="${charset:$idx:1}"
    done
    local content1="collision test 1 $(date)"
    local content2="collision test 2 $(date)"
    local url1
    url1=$(cpost -X POST "$NCLIP_URL/" -d "$content1" -H "X-Slug: $slug") || true
    if [[ -n "$url1" && "$url1" == http* ]]; then record_slug "$(basename "$url1")"; fi
    if [[ -z "$url1" || ! "$url1" =~ http ]]; then error "Failed to create first paste for collision test. Response: $url1"; return 1; fi
    local status
    status=$(cpost -s -o /dev/null -w "%{http_code}" -X POST "$NCLIP_URL/" -d "$content2" -H "X-Slug: $slug")
    if [[ "$status" == "400" ]]; then success "Slug collision test passed: server rejected duplicate slug"; return 0; else error "Slug collision test failed: expected 400, got $status"; return 1; fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cleanup_temp_files
    wait_for_nclip
    test_slug_collision
    cleanup_temp_files
fi

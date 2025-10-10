#!/bin/bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

test_not_found() {
    log "Testing 404 for non-existent paste..."
    local status
    local slug
    local data_dir="./data"
    local charset="ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
    local max_attempts=10
    local attempt=0
    while true; do
        slug=""
        for ((i=0; i<$SLUG_LENGTH; i++)); do
            idx=$(od -An -N2 -tu2 < /dev/urandom | awk '{print $1 % 32}')
            slug+="${charset:$idx:1}"
        done
        if [[ ! -f "$data_dir/$slug.json" && ! -f "$data_dir/$slug" ]]; then
            break
        fi
        ((attempt++))
        if [[ $attempt -ge $max_attempts ]]; then
            error "Could not find a non-existent slug after $max_attempts attempts"
            return 1
        fi
    done
    status=$(cget -s -o /dev/null -w "%{http_code}" "$NCLIP_URL/$slug")
    if [[ "$status" == "404" ]]; then
        success "404 test passed"
        return 0
    else
        error "404 test failed. Expected status 404, got: $status"
        return 1
    fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cleanup_temp_files
    wait_for_nclip
    test_not_found
    cleanup_temp_files
fi

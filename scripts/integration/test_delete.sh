#!/bin/bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
enable_cleanup_trap

# Test: Delete a paste via DELETE /{slug}
test_delete_paste() {
    local paste_url="$1"
    local slug
    slug=$(basename "$paste_url")
    local delete_url="$NCLIP_URL/$slug"
    log "Testing DELETE $delete_url"

    local response
    response=$(curl -sS "${CURL_AUTH_ARGS[@]}" -X DELETE "$delete_url")

    if [[ "$response" == *'"deleted":true'* ]] && [[ "$response" == *"$slug"* ]]; then
        success "Delete returned success: $response"
    else
        error "Delete response unexpected: $response"
        return 1
    fi

    # Verify paste is gone (should return 404)
    local http_code
    http_code=$(curl -sS -o /dev/null -w "%{http_code}" "${CURL_AUTH_ARGS[@]}" "$NCLIP_URL/raw/$slug")
    if [[ "$http_code" == "404" ]]; then
        success "Paste confirmed deleted (GET returns 404)"
    else
        error "Expected 404 after delete, got HTTP $http_code"
        return 1
    fi
}

# Test: Delete a non-existent paste (should return 404)
test_delete_not_found() {
    local slug="ZZZZZ"
    local delete_url="$NCLIP_URL/$slug"
    log "Testing DELETE non-existent paste: $delete_url"

    local http_code
    http_code=$(curl -sS -o /dev/null -w "%{http_code}" "${CURL_AUTH_ARGS[@]}" -X DELETE "$delete_url")
    if [[ "$http_code" == "404" ]]; then
        success "Delete non-existent paste correctly returns 404"
    else
        error "Expected 404 for non-existent paste, got HTTP $http_code"
        return 1
    fi
}

# Main
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cleanup_temp_files
    wait_for_nclip

    test_content="Delete integration test $(date)"
    log "Creating test paste for deletion..."
    paste_url=""
    if try_post paste_url "$NCLIP_URL/" "$test_content"; then
        if [[ -n "$paste_url" && "$paste_url" == http* ]]; then
            record_slug "$(basename "$paste_url")"
            success "Paste created: $paste_url"
            test_delete_paste "$paste_url"
        else
            error "Failed to create paste. Response: $paste_url"
            exit 1
        fi
    else
        error "Failed to create paste for delete test"
        exit 22
    fi

    test_delete_not_found

    cleanup_temp_files
fi

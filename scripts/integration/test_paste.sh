#!/bin/bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
enable_cleanup_trap

test_get_paste() {
    local paste_url="$1"
    local expected_content="$2"
    log "Testing paste retrieval from: $paste_url"
    local response
    response=$(cget_f "$paste_url")
    if [[ "$response" == "$expected_content" ]]; then
        success "Paste retrieval successful"
        return 0
    else
        error "Paste retrieval failed. Expected: '$expected_content', Got: '$response'"
        return 1
    fi
}

test_get_raw_paste() {
    local paste_url="$1"
    local expected_content="$2"
    local slug
    slug=$(basename "$paste_url")
    local raw_url="$NCLIP_URL/raw/$slug"
    log "Testing raw paste retrieval from: $raw_url"
    local response
    response=$(cget_f "$raw_url")
    if [[ "$response" == "$expected_content" ]]; then
        success "Raw paste retrieval successful"
        return 0
    else
        error "Raw paste retrieval failed. Expected: '$expected_content', Got: '$response'"
        return 1
    fi
}

test_get_metadata() {
    local paste_url="$1"
    local slug
    slug=$(basename "$paste_url")
    local meta_url="$NCLIP_URL/api/v1/meta/$slug"
    log "Testing metadata retrieval from: $meta_url"
    local response
    response=$(cget_f "$meta_url")
    if [[ "$response" == *"id"* ]] && [[ "$response" == *"created_at"* ]]; then
        success "Metadata retrieval successful: $response"
        return 0
    else
        error "Metadata retrieval failed. Response: $response"
        return 1
    fi
}

test_get_metadata_alias() {
    local paste_url="$1"
    local slug
    slug=$(basename "$paste_url")
    local json_url="$NCLIP_URL/json/$slug"
    log "Testing metadata alias from: $json_url"
    local response
    response=$(cget_f "$json_url")
    if [[ "$response" == *"id"* ]] && [[ "$response" == *"created_at"* ]]; then
        success "Metadata alias successful: $response"
        return 0
    else
        error "Metadata alias failed. Response: $response"
        return 1
    fi
}

# If executed directly, create a paste and run retrieval tests
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cleanup_temp_files
    wait_for_nclip
    test_content="Integration test content $(date)"
    log "Creating test paste with content: $test_content"
    paste_url=""
    if try_post paste_url "$NCLIP_URL/" "$test_content"; then
        if [[ -n "$paste_url" && "$paste_url" == http* ]]; then
            record_slug "$(basename "$paste_url")"
            success "Paste created successfully: $paste_url"
            test_get_paste "$paste_url" "$test_content"
            test_get_raw_paste "$paste_url" "$test_content"
            test_get_metadata "$paste_url"
            test_get_metadata_alias "$paste_url"
        else
            error "Failed to create paste. Response: $paste_url"
            exit 1
        fi
    else
        error "Failed to create initial paste after retries. Response: $paste_url"
        exit 22
    fi
    cleanup_temp_files
fi

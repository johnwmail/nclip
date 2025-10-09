#!/bin/bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

test_small_content_full_render() {
    log "Testing small content renders fully in HTML view"
    local test_content="Small content test"
    
    log "Uploading small text with explicit text/plain Content-Type"
    local paste_url
    # Use try_post so auth headers are included when set and retries are handled
    try_post paste_url "$NCLIP_URL/" "$test_content" -H "Content-Type: text/plain"
    
    if [[ -z "$paste_url" || "$paste_url" != http* ]]; then
        error "Failed to create paste for small content test. Response: $paste_url"
        return 1
    fi
    
    local slug
    slug=$(basename "$paste_url")
    record_slug "$slug"
    
    log "Fetching HTML view for small content: $paste_url"
    local html_response
    html_response=$(curl -s -H "User-Agent: Mozilla/5.0" "$paste_url")
    
    if [[ "$html_response" == *"$test_content"* ]]; then
        success "Small content rendered fully in HTML view"
    else
        error "Small content not found in HTML view"
        return 1
    fi
    
    # Verify it should NOT show preview indicator
    if [[ "$html_response" == *"Content Preview (Truncated)"* ]]; then
        error "Small content incorrectly shows preview indicator"
        return 1
    fi
    
    success "Small content test passed"
    return 0
}

test_large_content_preview() {
    log "Testing large content shows preview with truncation indicator"
    
    # Create content larger than default MaxRenderSize (262144 bytes)
    # Create 300KB of content
    local temp_file="/tmp/nclip_large_content_$$"
    dd if=/dev/zero bs=1024 count=300 2>/dev/null | tr '\0' 'A' > "$temp_file"
    
    log "Uploading large file (300KB) with explicit text/plain Content-Type"
    local paste_url
    # Use try_post to include auth and retry; provide data as @file and content-type
    try_post paste_url "$NCLIP_URL/" "@${temp_file}" -H "Content-Type: text/plain"
    
    if [[ -z "$paste_url" || "$paste_url" != http* ]]; then
        error "Failed to create paste for large content test. Response: $paste_url"
        rm -f "$temp_file"
        return 1
    fi
    
    local slug
    slug=$(basename "$paste_url")
    record_slug "$slug"
    
    log "Fetching HTML view for large content: $paste_url"
    local html_response
    html_response=$(curl -s -H "User-Agent: Mozilla/5.0" "$paste_url")
    
    # Should show preview indicator
    if [[ "$html_response" == *"Content Preview (Truncated)"* ]]; then
        success "Large content shows truncation indicator"
    else
        error "Large content missing truncation indicator"
        rm -f "$temp_file"
        return 1
    fi
    
    # Should show download link
    if [[ "$html_response" == *"Raw Data View"* ]]; then
        success "Large content shows raw data link"
    else
        error "Large content missing raw data link"
        rm -f "$temp_file"
        return 1
    fi
    
    # HTML view should NOT contain all 300KB (should be truncated)
    local html_size=${#html_response}
    if [[ $html_size -lt 300000 ]]; then
        success "HTML response is truncated (size: $html_size bytes)"
    else
        warn "HTML response seems large (size: $html_size bytes), might not be truncated"
    fi
    
    rm -f "$temp_file"
    success "Large content preview test passed"
    return 0
}

test_raw_always_full() {
    log "Testing raw endpoint always returns full content"
    
    # Create content larger than default MaxRenderSize
    local temp_file="/tmp/nclip_raw_full_$$"
    dd if=/dev/zero bs=1024 count=300 2>/dev/null | tr '\0' 'B' > "$temp_file"
    local expected_size=307200  # 300KB
    
    log "Uploading large file for raw test with explicit text/plain Content-Type"
    local paste_url
    try_post paste_url "$NCLIP_URL/" "@${temp_file}" -H "Content-Type: text/plain"
    
    if [[ -z "$paste_url" || "$paste_url" != http* ]]; then
        error "Failed to create paste for raw test. Response: $paste_url"
        rm -f "$temp_file"
        return 1
    fi
    
    local slug
    slug=$(basename "$paste_url")
    record_slug "$slug"
    
    log "Fetching raw content: $NCLIP_URL/raw/$slug"
    local raw_response
    raw_response=$(curl -s "$NCLIP_URL/raw/$slug")
    
    local raw_size=${#raw_response}
    if [[ $raw_size -eq $expected_size ]]; then
        success "Raw endpoint returned full content ($raw_size bytes)"
    else
        error "Raw endpoint size mismatch. Expected: $expected_size, Got: $raw_size"
        rm -f "$temp_file"
        return 1
    fi
    
    # Verify content is all 'B's
    local first_char="${raw_response:0:1}"
    local last_char="${raw_response: -1}"
    if [[ "$first_char" == "B" && "$last_char" == "B" ]]; then
        success "Raw content integrity verified"
    else
        error "Raw content integrity check failed"
        rm -f "$temp_file"
        return 1
    fi
    
    rm -f "$temp_file"
    success "Raw endpoint test passed"
    return 0
}

# Main execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cleanup_temp_files
    wait_for_nclip
    
    test_small_content_full_render || exit 1
    test_large_content_preview || exit 1
    test_raw_always_full || exit 1
    
    success "All preview tests passed"
    exit 0
fi

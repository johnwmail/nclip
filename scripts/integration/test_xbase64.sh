#!/bin/bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
enable_cleanup_trap

test_xbase64_true() {
    log "Testing X-Base64: true header..."
    local test_content="X-Base64 true test $(date)"
    local encoded
    encoded=$(echo -n "$test_content" | base64)
    
    local response
    response=$(echo -n "$encoded" | cpost -X POST "$NCLIP_URL/" \
        -H "X-Base64: true" \
        --data-binary @- || true)
    
    if [[ -n "$response" && "$response" == http* ]]; then
        record_slug "$(basename "$response")"
        log "Paste created with X-Base64: true: $response"
        
        local retrieved
        retrieved=$(cget_f "$response")
        if [[ "$retrieved" == "$test_content" ]]; then
            success "X-Base64: true decoded correctly"
            return 0
        else
            error "Content mismatch. Expected: '$test_content', Got: '$retrieved'"
            return 1
        fi
    else
        error "Failed to create paste with X-Base64: true. Response: $response"
        return 1
    fi
}

test_xbase64_false() {
    log "Testing X-Base64: false (should not decode)..."
    local test_content="X-Base64 false test"
    local encoded
    encoded=$(echo -n "$test_content" | base64)
    
    local response
    response=$(echo -n "$encoded" | cpost -X POST "$NCLIP_URL/" \
        -H "X-Base64: false" \
        --data-binary @- || true)
    
    if [[ -n "$response" && "$response" == http* ]]; then
        record_slug "$(basename "$response")"
        log "Paste created with X-Base64: false: $response"
        
        local retrieved
        retrieved=$(cget_f "$response")
        # Should retrieve the base64 encoded string, not decoded
        if [[ "$retrieved" == "$encoded" ]]; then
            success "X-Base64: false correctly did not decode"
            return 0
        else
            error "X-Base64: false should not decode. Expected: '$encoded', Got: '$retrieved'"
            return 1
        fi
    else
        error "Failed to create paste with X-Base64: false. Response: $response"
        return 1
    fi
}

test_xbase64_with_xttl() {
    log "Testing X-Base64 + X-TTL combination..."
    local test_content="Base64 + TTL test $(date)"
    local encoded
    encoded=$(echo -n "$test_content" | base64)
    
    local response
    response=$(echo -n "$encoded" | cpost -X POST "$NCLIP_URL/" \
        -H "X-Base64: true" \
        -H "X-TTL: 2h" \
        --data-binary @- || true)
    
    if [[ -n "$response" && "$response" == http* ]]; then
        record_slug "$(basename "$response")"
        log "Paste created with X-Base64 + X-TTL: $response"
        
        local retrieved
        retrieved=$(cget_f "$response")
        if [[ "$retrieved" == "$test_content" ]]; then
            success "X-Base64 + X-TTL combination working correctly"
            return 0
        else
            error "Content mismatch. Expected: '$test_content', Got: '$retrieved'"
            return 1
        fi
    else
        error "Failed to create paste with X-Base64 + X-TTL. Response: $response"
        return 1
    fi
}

test_xbase64_with_xslug() {
    log "Testing X-Base64 + X-Slug combination..."
    local test_content="Base64 + Slug test $(date)"
    local encoded
    encoded=$(echo -n "$test_content" | base64)
    local custom_slug="B64TST"
    
    local response
    response=$(echo -n "$encoded" | cpost -X POST "$NCLIP_URL/" \
        -H "X-Base64: true" \
        -H "X-Slug: $custom_slug" \
        --data-binary @- || true)
    
    if [[ -n "$response" && "$response" == http* ]]; then
        record_slug "$custom_slug"
        
        if echo "$response" | grep -q "$custom_slug"; then
            log "Paste created with custom slug: $response"
            
            local retrieved
            retrieved=$(cget_f "$response")
            if [[ "$retrieved" == "$test_content" ]]; then
                success "X-Base64 + X-Slug combination working correctly"
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
        error "Failed to create paste with X-Base64 + X-Slug. Response: $response"
        return 1
    fi
}

test_xbase64_with_xburn() {
    log "Testing X-Base64 + X-Burn combination..."
    local test_content="Base64 + Burn test $(date)"
    local encoded
    encoded=$(echo -n "$test_content" | base64)
    
    local response
    response=$(echo -n "$encoded" | cpost -X POST "$NCLIP_URL/" \
        -H "X-Base64: true" \
        -H "X-Burn: true" \
        --data-binary @- || true)
    
    if [[ -n "$response" && "$response" == http* ]]; then
        record_slug "$(basename "$response")"
        log "Paste created with X-Base64 + X-Burn: $response"
        
        local first_read
        first_read=$(cget_f "$response")
        if [[ "$first_read" == "$test_content" ]]; then
            success "First read with X-Base64 + X-Burn successful"
            
            local second_read_status
            second_read_status=$(cget -s -o /dev/null -w "%{http_code}" "$response")
            if [[ "$second_read_status" == "404" ]]; then
                success "X-Base64 + X-Burn combination working correctly"
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
        error "Failed to create paste with X-Base64 + X-Burn. Response: $response"
        return 1
    fi
}

test_xbase64_all_headers() {
    log "Testing X-Base64 with all custom headers combined..."
    local test_content="All headers test $(date)"
    local encoded
    encoded=$(echo -n "$test_content" | base64)
    local custom_slug="ALLHDR"
    
    local response
    response=$(echo -n "$encoded" | cpost -X POST "$NCLIP_URL/" \
        -H "X-Base64: true" \
        -H "X-Burn: true" \
        -H "X-TTL: 3h" \
        -H "X-Slug: $custom_slug" \
        --data-binary @- || true)
    
    if [[ -n "$response" && "$response" == http* ]]; then
        record_slug "$custom_slug"
        
        if echo "$response" | grep -q "$custom_slug"; then
            log "Paste created with all headers: $response"
            
            local first_read
            first_read=$(cget_f "$response")
            if [[ "$first_read" == "$test_content" ]]; then
                success "First read with all headers successful"
                
                local second_read_status
                second_read_status=$(cget -s -o /dev/null -w "%{http_code}" "$response")
                if [[ "$second_read_status" == "404" ]]; then
                    success "All headers combination working correctly (X-Base64 + X-Burn + X-TTL + X-Slug)"
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

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cleanup_temp_files
    wait_for_nclip
    
    log "=== X-Base64 Header Integration Tests ==="
    test_xbase64_true || exit 1
    test_xbase64_false || exit 1
    test_xbase64_with_xttl || exit 1
    test_xbase64_with_xslug || exit 1
    test_xbase64_with_xburn || exit 1
    test_xbase64_all_headers || exit 1
    
    success "✓ All X-Base64 header tests passed!"
    cleanup_temp_files
fi

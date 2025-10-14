#!/usr/bin/env bash
# Integration test for X-Burn header functionality

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

test_xburn_header() {
    log "=== X-Burn Header Tests ==="
    
    # Test 1: X-Burn: true on regular route
    log "Test 1: X-Burn: true header on / route"
    local content="Burn this message - Test 1"
    
    response=$(echo -n "$content" | curl -s -X POST "$NCLIP_URL/" \
        -H "X-Burn: true" \
        --data-binary @-)
    
    if echo "$response" | grep -q "http"; then
        slug=$(echo "$response" | grep -o '[A-HJ-NP-Z2-9]\{5\}$')
        success "Upload with X-Burn: true succeeded: $slug"
        
        # First read should succeed
        first_read=$(curl -s "$NCLIP_URL/$slug")
        if echo "$first_read" | grep -q "$content"; then
            success "First read succeeded"
        else
            error "First read failed or content mismatch"
            return 1
        fi
        
        # Second read should fail (404)
        second_read=$(curl -s -w "%{http_code}" -o /dev/null "$NCLIP_URL/$slug")
        if [ "$second_read" = "404" ]; then
            success "Second read returned 404 (paste burned)"
        else
            error "Second read should return 404, got: $second_read"
            return 1
        fi
    else
        error "Upload failed: $response"
        return 1
    fi
    
    # Test 2: X-Burn: 1 variant
    log "Test 2: X-Burn: 1 header"
    content="Burn this message - Test 2"
    
    response=$(echo -n "$content" | curl -s -X POST "$NCLIP_URL/" \
        -H "X-Burn: 1" \
        --data-binary @-)
    
    if echo "$response" | grep -q "http"; then
        slug=$(echo "$response" | grep -o '[A-HJ-NP-Z2-9]\{5\}$')
        success "Upload with X-Burn: 1 succeeded: $slug"
        
        # Verify it burns
        curl -s "$NCLIP_URL/$slug" > /dev/null
        second_read=$(curl -s -w "%{http_code}" -o /dev/null "$NCLIP_URL/$slug")
        if [ "$second_read" = "404" ]; then
            success "Paste with X-Burn: 1 burned correctly"
        else
            error "Should burn after first read"
            return 1
        fi
    else
        error "Upload failed: $response"
        return 1
    fi
    
    # Test 3: X-Burn: yes variant
    log "Test 3: X-Burn: yes header"
    content="Burn this message - Test 3"
    
    response=$(echo -n "$content" | curl -s -X POST "$NCLIP_URL/" \
        -H "X-Burn: yes" \
        --data-binary @-)
    
    if echo "$response" | grep -q "http"; then
        slug=$(echo "$response" | grep -o '[A-HJ-NP-Z2-9]\{5\}$')
        success "Upload with X-Burn: yes succeeded: $slug"
        
        # Verify it burns
        curl -s "$NCLIP_URL/$slug" > /dev/null
        second_read=$(curl -s -w "%{http_code}" -o /dev/null "$NCLIP_URL/$slug")
        if [ "$second_read" = "404" ]; then
            success "Paste with X-Burn: yes burned correctly"
        else
            error "Should burn after first read"
            return 1
        fi
    else
        error "Upload failed: $response"
        return 1
    fi
    
    # Test 4: No X-Burn header (should NOT burn)
    log "Test 4: Upload without X-Burn header (should not burn)"
    content="Keep this message - Test 4"
    
    response=$(echo -n "$content" | curl -s -X POST "$NCLIP_URL/" \
        --data-binary @-)
    
    if echo "$response" | grep -q "http"; then
        slug=$(echo "$response" | grep -o '[A-HJ-NP-Z2-9]\{5\}$')
        success "Upload without X-Burn succeeded: $slug"
        
        # Multiple reads should work
        curl -s "$NCLIP_URL/raw/$slug" > /dev/null
        second_read=$(curl -s "$NCLIP_URL/raw/$slug")
        if echo "$second_read" | grep -q "$content"; then
            success "Multiple reads work (not burned)"
        else
            error "Should not burn without X-Burn header"
            return 1
        fi
    else
        error "Upload failed: $response"
        return 1
    fi
    
    # Test 5: X-Burn: false (should NOT burn)
    log "Test 5: X-Burn: false header (should not burn)"
    content="Keep this too - Test 5"
    
    response=$(echo -n "$content" | curl -s -X POST "$NCLIP_URL/" \
        -H "X-Burn: false" \
        --data-binary @-)
    
    if echo "$response" | grep -q "http"; then
        slug=$(echo "$response" | grep -o '[A-HJ-NP-Z2-9]\{5\}$')
        success "Upload with X-Burn: false succeeded: $slug"
        
        # Multiple reads should work
        curl -s "$NCLIP_URL/raw/$slug" > /dev/null
        second_read=$(curl -s "$NCLIP_URL/raw/$slug")
        if echo "$second_read" | grep -q "$content"; then
            success "X-Burn: false does not burn"
        else
            error "X-Burn: false should not burn paste"
            return 1
        fi
    else
        error "Upload failed: $response"
        return 1
    fi
    
    # Test 6: X-Burn: 0 (should NOT burn)
    log "Test 6: X-Burn: 0 header (should not burn)"
    content="Keep this three - Test 6"
    
    response=$(echo -n "$content" | curl -s -X POST "$NCLIP_URL/" \
        -H "X-Burn: 0" \
        --data-binary @-)
    
    if echo "$response" | grep -q "http"; then
        slug=$(echo "$response" | grep -o '[A-HJ-NP-Z2-9]\{5\}$')
        success "Upload with X-Burn: 0 succeeded: $slug"
        
        # Multiple reads should work
        curl -s "$NCLIP_URL/raw/$slug" > /dev/null
        second_read=$(curl -s "$NCLIP_URL/raw/$slug")
        if echo "$second_read" | grep -q "$content"; then
            success "X-Burn: 0 does not burn"
        else
            error "X-Burn: 0 should not burn paste"
            return 1
        fi
    else
        error "Upload failed: $response"
        return 1
    fi
}

test_xburn_with_base64() {
    log "=== X-Burn + Base64 Combined Tests ==="
    
    # Test 1: X-Burn header + X-Base64: true
    log "Test 1: X-Burn + base64 encoding via headers"
    local content="Burn this base64 message"
    local encoded=$(echo -n "$content" | base64)
    
    response=$(echo -n "$encoded" | curl -s -X POST "$NCLIP_URL/" \
        -H "X-Burn: true" \
        -H "X-Base64: true" \
        --data-binary @-)
    
    if echo "$response" | grep -q "http"; then
        slug=$(echo "$response" | grep -o '[A-HJ-NP-Z2-9]\{5\}$')
        success "Upload with X-Burn + base64 succeeded: $slug"
        
        # Verify content is decoded (curl gets raw content, not HTML)
        first_read=$(curl -s "$NCLIP_URL/$slug")
        if echo "$first_read" | grep -q "$content"; then
            success "Content decoded correctly"
        else
            error "Content not decoded: content not found in response"
            return 1
        fi
        
        # Verify it burns
        second_read=$(curl -s -w "%{http_code}" -o /dev/null "$NCLIP_URL/$slug")
        if [ "$second_read" = "404" ]; then
            success "Base64 paste burned correctly"
        else
            error "Should burn after first read, got status: $second_read"
            return 1
        fi
    else
        error "Upload failed: $response"
        return 1
    fi
    
    # Test 2: X-Burn header + /base64 route
    log "Test 2: X-Burn header + /base64 route"
    content="Burn via route and header"
    encoded=$(echo -n "$content" | base64)
    
    response=$(echo -n "$encoded" | curl -s -X POST "$NCLIP_URL/base64" \
        -H "X-Burn: true" \
        --data-binary @-)
    
    if echo "$response" | grep -q "http"; then
        slug=$(echo "$response" | grep -o '[A-HJ-NP-Z2-9]\{5\}$')
        success "Upload via /base64 route with X-Burn succeeded: $slug"
        
        # Verify content and burn (curl gets raw content from /:slug)
        first_read=$(curl -s "$NCLIP_URL/$slug")
        if echo "$first_read" | grep -q "$content"; then
            success "Content decoded correctly"
        else
            error "Content mismatch"
            return 1
        fi
        
        second_read=$(curl -s -w "%{http_code}" -o /dev/null "$NCLIP_URL/$slug")
        if [ "$second_read" = "404" ]; then
            success "Burned correctly via /base64 + X-Burn"
        else
            error "Should burn, got status: $second_read"
            return 1
        fi
    else
        error "Upload failed: $response"
        return 1
    fi
}

test_burn_route_backward_compat() {
    log "=== /burn/ Route Backward Compatibility ==="
    
    # Test: Old /burn/ route should still work
    log "Test: /burn/ route without X-Burn header"
    local content="Burn via route only"
    
    response=$(echo -n "$content" | curl -s -X POST "$NCLIP_URL/burn/" \
        --data-binary @-)
    
    if echo "$response" | grep -q "http"; then
        slug=$(echo "$response" | grep -o '[A-HJ-NP-Z2-9]\{5\}$')
        success "/burn/ route still works: $slug"
        
        # Verify it burns
        curl -s "$NCLIP_URL/$slug" > /dev/null
        second_read=$(curl -s -w "%{http_code}" -o /dev/null "$NCLIP_URL/$slug")
        if [ "$second_read" = "404" ]; then
            success "/burn/ route burned correctly (backward compatible)"
        else
            error "/burn/ route should burn paste"
            return 1
        fi
    else
        error "/burn/ route upload failed: $response"
        return 1
    fi
}

# Run all tests
main() {
    log "Starting X-Burn header integration tests..."
    log "BASE_URL: $NCLIP_URL"
    echo
    
    test_xburn_header
    echo
    
    test_xburn_with_base64
    echo
    
    test_burn_route_backward_compat
    echo
    
    success "✓ All X-Burn header tests passed!"
}

main "$@"

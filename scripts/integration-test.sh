#!/bin/bash
set -euo pipefail


# Integration test script for nclip (S3/filesystem or any backend)
# This script tests all major API endpoints to ensure they work correctly, regardless of storage backend.


# Configuration
NCLIP_URL="${NCLIP_URL:-http://localhost:8080}"
SLUG_LENGTH="${SLUG_LENGTH:-5}"
TEST_TIMEOUT=30
RETRY_DELAY=2
MAX_RETRIES=15

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

# Wait for nclip to be ready
wait_for_nclip() {
    log "Waiting for nclip to be ready at $NCLIP_URL..."
    
    for i in $(seq 1 $MAX_RETRIES); do
        if curl -f -s "$NCLIP_URL/health" > /dev/null 2>&1; then
            success "nclip is ready!"
            return 0
        fi
        
        log "Attempt $i/$MAX_RETRIES: nclip not ready yet, waiting ${RETRY_DELAY}s..."
        sleep $RETRY_DELAY
    done
    
    error "nclip failed to become ready after $MAX_RETRIES attempts"
    return 1
}

# Test health endpoint
test_health() {
    log "Testing health endpoint..."
    
    local response
    response=$(curl -f -s "$NCLIP_URL/health")
    
    if [[ "$response" == *"healthy"* ]] || [[ "$response" == *"ok"* ]] || [[ "$response" == *"OK"* ]]; then
        success "Health check passed: $response"
        return 0
    else
        error "Health check failed: $response"
        return 1
    fi
}

# Test paste retrieval (GET /:slug)
test_get_paste() {
    local paste_url="$1"
    local expected_content="$2"
    
    log "Testing paste retrieval from: $paste_url"
    
    local response
    response=$(curl -f -s "$paste_url")
    
    if [[ "$response" == "$expected_content" ]]; then
        success "Paste retrieval successful"
        return 0
    else
        error "Paste retrieval failed. Expected: '$expected_content', Got: '$response'"
        return 1
    fi
}

# Test raw paste retrieval (GET /raw/:slug)
test_get_raw_paste() {
    local paste_url="$1"
    local expected_content="$2"
    
    # Extract slug from URL
    local slug
    slug=$(basename "$paste_url")
    local raw_url="$NCLIP_URL/raw/$slug"
    
    log "Testing raw paste retrieval from: $raw_url"
    
    local response
    response=$(curl -f -s "$raw_url")
    
    if [[ "$response" == "$expected_content" ]]; then
        success "Raw paste retrieval successful"
        return 0
    else
        error "Raw paste retrieval failed. Expected: '$expected_content', Got: '$response'"
        return 1
    fi
}

# Test metadata API (GET /api/v1/meta/:slug)
test_get_metadata() {
    local paste_url="$1"
    
    # Extract slug from URL
    local slug
    slug=$(basename "$paste_url")
    local meta_url="$NCLIP_URL/api/v1/meta/$slug"
    
    log "Testing metadata retrieval from: $meta_url"
    
    local response
    response=$(curl -f -s "$meta_url")
    
    if [[ "$response" == *"id"* ]] && [[ "$response" == *"created_at"* ]]; then
        success "Metadata retrieval successful: $response"
        return 0
    else
        error "Metadata retrieval failed. Response: $response"
        return 1
    fi
}

# Test metadata alias (GET /json/:slug)
test_get_metadata_alias() {
    local paste_url="$1"
    
    # Extract slug from URL
    local slug
    slug=$(basename "$paste_url")
    local json_url="$NCLIP_URL/json/$slug"
    
    log "Testing metadata alias from: $json_url"
    
    local response
    response=$(curl -f -s "$json_url")
    
    if [[ "$response" == *"id"* ]] && [[ "$response" == *"created_at"* ]]; then
        success "Metadata alias successful: $response"
        return 0
    else
        error "Metadata alias failed. Response: $response"
        return 1
    fi
}

# Test burn-after-read functionality (POST /burn/)
test_burn_after_read() {
    log "Testing burn-after-read functionality..."
    
    local test_content="Burn after read test $(date)"
    local response
    
    response=$(curl -f -s -X POST "$NCLIP_URL/burn/" -d "$test_content")
    
    if [[ -n "$response" ]] && [[ "$response" == http* ]]; then
        log "Burn paste created: $response"
        
        # Read the paste once
        local first_read
        first_read=$(curl -f -s "$response")
        
        if [[ "$first_read" == "$test_content" ]]; then
            success "First read of burn paste successful"
            
            # Try to read again - should fail
            local second_read_status
            second_read_status=$(curl -s -o /dev/null -w "%{http_code}" "$response")
            
            if [[ "$second_read_status" == "404" ]]; then
                success "Burn-after-read functionality working correctly"
                return 0
            else
                error "Burn paste still accessible after first read (status: $second_read_status)"
                return 1
            fi
        else
            error "First read of burn paste failed. Expected: '$test_content', Got: '$first_read'"
            return 1
        fi
    else
        error "Failed to create burn paste. Response: $response"
        return 1
    fi
}

# Test 404 for non-existent paste
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
    status=$(curl -s -o /dev/null -w "%{http_code}" "$NCLIP_URL/$slug")
    if [[ "$status" == "404" ]]; then
        success "404 test passed"
        return 0
    else
        error "404 test failed. Expected status 404, got: $status"
        return 1
    fi
}

# Main test function
run_integration_tests() {
    log "Starting nclip integration tests..."
    log "Target URL: $NCLIP_URL"
    echo
    
    # Wait for service to be ready
    if ! wait_for_nclip; then
        error "nclip service not ready, aborting tests"
        return 1
    fi
    echo
    
    local failed_tests=0
    
    # Test health endpoint
    if ! test_health; then
        ((failed_tests++))
    fi
    echo
    
    # Test paste creation and retrieval
    local test_content="Integration test content $(date)"
    
    log "Creating test paste with content: $test_content"
    local paste_url
    paste_url=$(curl -f -s -X POST "$NCLIP_URL/" -d "$test_content")
    
    if [[ -n "$paste_url" ]] && [[ "$paste_url" == http* ]]; then
        success "Paste created successfully: $paste_url"
        
        # Test paste retrieval
        if ! test_get_paste "$paste_url" "$test_content"; then
            ((failed_tests++))
        fi
        echo
        
        # Test raw paste retrieval
        if ! test_get_raw_paste "$paste_url" "$test_content"; then
            ((failed_tests++))
        fi
        echo
        
        # Test metadata retrieval
        if ! test_get_metadata "$paste_url"; then
            ((failed_tests++))
        fi
        echo
        
        # Test metadata alias
        if ! test_get_metadata_alias "$paste_url"; then
            ((failed_tests++))
        fi
        echo
    else
        error "Failed to create paste. Response: $paste_url"
        ((failed_tests++))
        echo
    fi
    
    # Test burn-after-read
    if ! test_burn_after_read; then
        ((failed_tests++))
    fi
    echo
    
    # Test 404
    if ! test_not_found; then
        ((failed_tests++))
    fi
    echo
    
    # Summary
    if [[ $failed_tests -eq 0 ]]; then
        success "All integration tests passed! ✨"
        return 0
    else
        error "$failed_tests test(s) failed!"
        return 1
    fi
}

# Run tests if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    run_integration_tests
fi
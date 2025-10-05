#!/bin/bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

test_buffer_size_limit() {
    log "Testing buffer size limit enforcement..."
    log "Attempting to upload 5MB+1 byte content (exceeds 5MB limit)..."
    local temp_file
    temp_file=$(mktemp)
    dd if=/dev/zero bs=1M count=5 2>/dev/null | tr '\0' 'X' > "$temp_file"
    echo -n 'X' >> "$temp_file"
    local file_size
    file_size=$(stat -c%s "$temp_file" 2>/dev/null || wc -c < "$temp_file")
    log "Created test file with size: $file_size bytes"
    local response
    response=$(cpost -s --max-time 30 -w "\n%{http_code}" -X POST "$NCLIP_URL/" --data-binary "@$temp_file" 2>/dev/null || true)
    local status
    status=$(echo "$response" | tail -n1)
    response=$(echo "$response" | sed '$d')
    rm -f "$temp_file"
    if [[ "$status" == "413" ]]; then
        if [[ "$response" == *"content too large"* || "$response" == *"too large"* ]]; then success "Buffer size limit correctly enforced for direct POST: $status - $response"; else error "Buffer size limit returned 413 but unexpected message: $response"; return 1; fi
    else
        error "Buffer size limit not enforced for direct POST. Expected 413, got $status. Response: $response"; return 1
    fi

    log "Testing multipart file upload size limit..."
    temp_file=$(mktemp)
    dd if=/dev/zero bs=1M count=5 2>/dev/null | tr '\0' 'X' > "$temp_file"
    echo -n "A" >> "$temp_file"
    response=$(cpost -s --max-time 30 -w "\n%{http_code}" -X POST "$NCLIP_URL/" -F "file=@$temp_file" 2>/dev/null || true)
    status=$(echo "$response" | tail -n1)
    response=$(echo "$response" | sed '$d')
    rm -f "$temp_file"
    if [[ "$status" == "413" ]]; then
        if [[ "$response" == *"content too large"* || "$response" == *"too large"* ]]; then success "Buffer size limit correctly enforced for multipart upload: $status - $response"; else error "Buffer size limit returned 413 but unexpected message: $response"; return 1; fi
    else
        error "Buffer size limit not enforced for multipart upload. Expected 413, got $status. Response: $response"; return 1
    fi

    success "Buffer size limit tests passed"
    return 0
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cleanup_temp_files
    wait_for_nclip
    test_buffer_size_limit
    cleanup_temp_files
fi

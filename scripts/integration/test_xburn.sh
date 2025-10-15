#!/bin/bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
enable_cleanup_trap

test_xburn_after_read() {
    log "Testing X-Burn after read functionality..."
    local test_content="X-Burn after read test $(date)"
    local response
    response=$(cpost -X POST "$NCLIP_URL/" -d "$test_content" -H "X-Burn: true" || true)
    if [[ -n "$response" && "$response" == http* ]]; then
        record_slug "$(basename "$response")"
    fi
    if [[ -n "$response" ]] && [[ "$response" == http* ]]; then
        log "X-Burn paste created: $response"
        local first_read
        first_read=$(cget_f "$response" )
        if [[ "$first_read" == "$test_content" ]]; then
            success "First read of X-Burn paste successful"
            local second_read_status
            # Use cget to honor auth headers; get only the HTTP status code
            second_read_status=$(cget -s -o /dev/null -w "%{http_code}" "$response")
            if [[ "$second_read_status" == "404" ]]; then
                success "X-Burn after read functionality working correctly"
                return 0
            else
                error "X-Burn paste still accessible after first read (status: $second_read_status)"
                return 1
            fi
        else
            error "First read of X-Burn paste failed. Expected: '$test_content', Got: '$first_read'"
            return 1
        fi
    else
        error "Failed to create X-Burn paste. Response: $response"
        return 1
    fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cleanup_temp_files
    wait_for_nclip
    test_xburn_after_read
    cleanup_temp_files
fi

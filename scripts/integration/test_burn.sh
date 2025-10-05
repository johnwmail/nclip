#!/bin/bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

test_burn_after_read() {
    log "Testing burn-after-read functionality..."
    local test_content="Burn after read test $(date)"
    local response
    response=$(cpost -X POST "$NCLIP_URL/burn/" -d "$test_content" || true)
    if [[ -n "$response" && "$response" == http* ]]; then
        record_slug "$(basename "$response")"
    fi
    if [[ -n "$response" ]] && [[ "$response" == http* ]]; then
        log "Burn paste created: $response"
        local first_read
        first_read=$(cget_f "$response" )
        if [[ "$first_read" == "$test_content" ]]; then
            success "First read of burn paste successful"
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

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cleanup_temp_files
    wait_for_nclip
    test_burn_after_read
    cleanup_temp_files
fi

#!/bin/bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

test_text_file_extensions() {
    log "Testing text file extensions in Content-Disposition..."
    local failed_tests=0
    local response

    # Helper to post and check header for a given body and expected extension
    post_and_check() {
        local body="$1"; shift
        local content_type="$1"; shift
        local expected_ext="$1"; shift
        if try_post response "$NCLIP_URL/" "$body" -H "Content-Type: ${content_type}"; then
            :
        fi
        if [[ -n "$response" && "$response" == http* ]]; then
            record_slug "$(basename "$response")"
            local slug
            slug=$(basename "$response")
            local raw_url="$NCLIP_URL/raw/$slug"
            local header
            header=$(cget -s -D - "$raw_url" -o /dev/null | grep -i "Content-Disposition")
            if [[ "$header" == *"$slug.${expected_ext}"* ]]; then
                success "${content_type} extension correctly appended: .${expected_ext}"
            else
                error "${content_type} extension NOT appended correctly. Header: $header"
                ((failed_tests++))
            fi
        else
            error "Failed to upload ${content_type} content. Response: $response"
            ((failed_tests++))
        fi
    }

    post_and_check "Hello World" "text/plain" "txt"
    post_and_check "<html><body>Hello</body></html>" "text/html" "html"
    post_and_check "console.log('hello');" "text/javascript" "js"
    post_and_check '{"name":"hello"}' "application/json" "json"

    if [[ $failed_tests -eq 0 ]]; then
        success "All text file extension tests passed"
        return 0
    else
        error "$failed_tests text file extension test(s) failed"
        return 1
    fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cleanup_temp_files
    wait_for_nclip
    test_text_file_extensions
    cleanup_temp_files
fi

#!/bin/bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

test_text_file_extensions() {
    log "Testing text file extensions in Content-Disposition..."
    local failed_tests=0
    # text/plain -> .txt
    log "Testing text/plain -> .txt"
    local response
    response=$(cpost -X POST "$NCLIP_URL/" -d "Hello World" -H "Content-Type: text/plain")
    if [[ -n "$response" && "$response" == http* ]]; then record_slug "$(basename "$response")"; fi
    if [[ -z "$response" || ! "$response" =~ http ]]; then
        error "Failed to upload text/plain content. Response: $response"
        ((failed_tests++))
    else
        local slug
        slug=$(basename "$response")
        local raw_url="$NCLIP_URL/raw/$slug"
        local header
        header=$(curl -s -D - "$raw_url" -o /dev/null | grep -i "Content-Disposition")
        if [[ "$header" == *"$slug.txt"* ]]; then success "text/plain extension correctly appended: .txt"; else error "text/plain extension NOT appended correctly. Header: $header"; ((failed_tests++)); fi
    fi

    # text/html -> .html
    log "Testing text/html -> .html"
    response=$(cpost -X POST "$NCLIP_URL/" -d "<html><body>Hello</body></html>" -H "Content-Type: text/html")
    if [[ -n "$response" && "$response" == http* ]]; then record_slug "$(basename "$response")"; fi
    if [[ -z "$response" || ! "$response" =~ http ]]; then error "Failed to upload text/html content. Response: $response"; ((failed_tests++)); else slug=$(basename "$response"); raw_url="$NCLIP_URL/raw/$slug"; header=$(curl -s -D - "$raw_url" -o /dev/null | grep -i "Content-Disposition"); if [[ "$header" == *"$slug.html"* ]]; then success "text/html extension correctly appended: .html"; else error "text/html extension NOT appended correctly. Header: $header"; ((failed_tests++)); fi; fi

    # text/javascript -> .js
    log "Testing text/javascript -> .js"
    response=$(cpost -X POST "$NCLIP_URL/" -d "console.log('hello');" -H "Content-Type: text/javascript")
    if [[ -n "$response" && "$response" == http* ]]; then record_slug "$(basename "$response")"; fi
    if [[ -z "$response" || ! "$response" =~ http ]]; then error "Failed to upload text/javascript content. Response: $response"; ((failed_tests++)); else slug=$(basename "$response"); raw_url="$NCLIP_URL/raw/$slug"; header=$(curl -s -D - "$raw_url" -o /dev/null | grep -i "Content-Disposition"); if [[ "$header" == *"$slug.js"* ]]; then success "text/javascript extension correctly appended: .js"; else error "text/javascript extension NOT appended correctly. Header: $header"; ((failed_tests++)); fi; fi

    # application/json -> .json
    log "Testing application/json -> .json"
    response=$(cpost -X POST "$NCLIP_URL/" -d '{"name":"hello"}' -H "Content-Type: application/json")
    if [[ -n "$response" && "$response" == http* ]]; then record_slug "$(basename "$response")"; fi
    if [[ -z "$response" || ! "$response" =~ http ]]; then error "Failed to upload application/json content. Response: $response"; ((failed_tests++)); else slug=$(basename "$response"); raw_url="$NCLIP_URL/raw/$slug"; header=$(curl -s -D - "$raw_url" -o /dev/null | grep -i "Content-Disposition"); if [[ "$header" == *"$slug.json"* ]]; then success "application/json extension correctly appended: .json"; else error "application/json extension NOT appended correctly. Header: $header"; ((failed_tests++)); fi; fi

    if [[ $failed_tests -eq 0 ]]; then success "All text file extension tests passed"; return 0; else error "$failed_tests text file extension test(s) failed"; return 1; fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cleanup_temp_files
    wait_for_nclip
    test_text_file_extensions
    cleanup_temp_files
fi

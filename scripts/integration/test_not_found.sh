#!/bin/bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
enable_cleanup_trap

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
    status=$(cget -s -o /dev/null -w "%{http_code}" "$NCLIP_URL/$slug")
    if [[ "$status" == "404" ]]; then
        success "404 test passed"
        # Also verify browser-like requests return HTML
        test_not_found_browser "$slug"
        return 0
    else
        error "404 test failed. Expected status 404, got: $status"
        return 1
    fi
}

test_not_found_browser() {
    log "Testing browser-like request returns HTML for non-existent paste..."
    local slug="$1"
    local ua="Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36"

    # Capture headers only to inspect Content-Type
    local headers
    headers=$(curl -sS "${CURL_AUTH_ARGS[@]}" -A "$ua" -H "Accept: text/html" -D - -o /dev/null "$NCLIP_URL/$slug" || true)
    # Normalize line endings and extract Content-Type value
    local content_type
    content_type=$(echo "$headers" | tr -d '\r' | awk -F": " '/^Content-Type:/ {print $2; exit}' | cut -d';' -f1 || true)

    # Fetch body to assert it's HTML-ish
    local body
    body=$(curl -sS "${CURL_AUTH_ARGS[@]}" -A "$ua" -H "Accept: text/html" "$NCLIP_URL/$slug" || true)

    if [[ "$content_type" == "text/html" ]] || echo "$body" | grep -qiE '<!doctype html|<html'; then
        success "Browser 404 HTML test passed"
        return 0
    else
        error "Browser 404 test failed. Expected HTML content-type or HTML body; got Content-Type='$content_type'"
        return 1
    fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cleanup_temp_files
    wait_for_nclip
    test_not_found
    cleanup_temp_files
fi

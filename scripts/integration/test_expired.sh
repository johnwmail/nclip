#!/bin/bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

test_expired_paste() {
    log "Testing expired paste behavior (creation only)..."
    local test_content="Expired paste test $(date)"
    local ttl="1h"
    local paste_url
    paste_url=$(cpost -X POST "$NCLIP_URL/" -d "$test_content" -H "X-TTL: $ttl") || true
    if [[ -z "$paste_url" || ! "$paste_url" =~ http ]]; then error "Failed to create paste with TTL. Response: $paste_url"; return 1; fi
    log "Paste created with TTL: $paste_url"
    success "Expired paste test (creation only) passed. Manual expiry not tested."
    return 0
}

test_expired_paste_manual() {
    log "Testing expired paste behavior with manual file injection and on-disk verification..."
    # Create a real paste via the API so we have a valid slug that the server accepts
    log "Creating a real paste to get a valid slug..."
    local paste_url
    paste_url=$(cpost -X POST "$NCLIP_URL/" -d "manual expired content" ) || true
    if [[ -z "$paste_url" || ! "$paste_url" =~ http ]]; then
        error "Failed to create paste for manual expired test. Response: $paste_url"
        return 1
    fi
    local slug
    slug=${paste_url##*/}

    # Use date to emit RFC3339-like timestamps in UTC with Z suffix.
    # We include seconds and nanoseconds where supported; if nanoseconds are not available
    # the timestamp will still be acceptable to Go's time parser.
    local now
    if date -u +%Y-%m-%dT%H:%M:%S.%NZ >/dev/null 2>&1; then
        now=$(date -u +%Y-%m-%dT%H:%M:%S.%NZ)
    else
        now=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    fi
    local expired
    if date -u -d "-1 day" +%Y-%m-%dT%H:%M:%S.%NZ >/dev/null 2>&1; then
        expired=$(date -u -d "-1 day" +%Y-%m-%dT%H:%M:%S.%NZ)
    else
        expired=$(date -u -d "-1 day" +%Y-%m-%dT%H:%M:%SZ)
    fi

    # Allow overriding data dir for CI/debugging
    local dir="${NCLIP_DATA_DIR:-./data}"
    mkdir -p "$dir"

    local meta_file="$dir/${slug}.json"
    local content_file="$dir/${slug}"

    # Overwrite the metadata to make it expired (server will read this and delete)
    cat > "$meta_file" <<EOF
{
  "id": "$slug",
  "created_at": "$now",
  "expires_at": "$expired",
  "size": 4,
  "content_type": "text/plain; charset=utf-8",
  "burn_after_read": false,
  "read_count": 0
}
EOF
    # Ensure content file exists; the server created it during POST but write if not present
    if [[ ! -e "$content_file" ]]; then
        echo "data" > "$content_file"
    fi
    log "Injected expired paste metadata (overwrote): $meta_file"
    log "Ensured content file exists: $content_file"

    # Trigger an HTTP GET which should cause the server to detect expiry and delete files
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" "$NCLIP_URL/$slug" || true)
    if [[ "$status" != "404" && "$status" != "400" && "$status" != "410" ]]; then
        error "Unexpected HTTP status for expired paste GET: $status"
        # show directory contents to aid debugging
        ls -la "$dir" || true
        if [[ -f "$meta_file" ]]; then
            echo "--- BEGIN META FILE ($meta_file) ---"
            cat "$meta_file" || true
            echo "--- END META FILE ---"
        fi
        rm -f "$meta_file" "$content_file" || true
        return 1
    fi

    # Verify metadata file was removed
    if [[ -e "$meta_file" ]]; then
        error "Metadata file still exists after expired GET: $meta_file"
        echo "--- BEGIN META FILE ($meta_file) ---"
        cat "$meta_file" || true
        echo "--- END META FILE ---"
        ls -la "$meta_file" || true
        rm -f "$meta_file" "$content_file" || true
        return 1
    fi

    # Verify content file was removed
    if [[ -e "$content_file" ]]; then
        error "Content file still exists after expired GET: $content_file"
        ls -la "$content_file" || true
        rm -f "$content_file" || true
        return 1
    fi

    success "Manual expired paste test passed: metadata and content removed from $dir"
    return 0
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cleanup_temp_files
    wait_for_nclip
    test_expired_paste
    test_expired_paste_manual
    cleanup_temp_files
fi

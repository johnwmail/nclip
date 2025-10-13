#!/usr/bin/env bash
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$DIR/lib.sh"

enable_cleanup_trap

log "Starting size mismatch integration test"

wait_for_nclip

# Create a temporary file to upload
TMPFILE=$(mktemp /tmp/nclip_test_content.XXXX)
echo "hello world" > "$TMPFILE"

# POST the file and capture response (upload handler returns URL or slug in body)
OUT=""
if try_post OUT "$NCLIP_URL/" "@${TMPFILE}" -H "Content-Type: text/plain"; then
    log "Upload succeeded"
else
    error "Upload failed: $OUT"
    exit 2
fi

# The upload handler responds with a URL or slug; try to extract slug (last path segment)
SLUG=$(echo "$OUT" | awk -F'/' '{print $NF}' | tr -d '\r\n')
if [[ -z "$SLUG" ]]; then
    error "Failed to parse slug from upload response: $OUT"
    exit 2
fi
record_slug "$SLUG"
log "Uploaded slug: $SLUG"

# Ensure file exists in data dir
if [[ ! -f "${NCLIP_DATA_DIR}/${SLUG}" ]]; then
    warn "Content file not found in data dir: ${NCLIP_DATA_DIR}/${SLUG}"
fi

# Corrupt/truncate the stored file to trigger size mismatch
log "Truncating stored file to 1 byte to simulate corruption"
truncate -s 1 "${NCLIP_DATA_DIR}/${SLUG}"

# Try to GET the paste (CLI style) and check for size_mismatch in JSON
init_auth
RESP=$(cget -s -S -w "\n%{http_code}" "$NCLIP_URL/${SLUG}" || true)
HTTP_STATUS=$(echo "$RESP" | tail -n1 || true)
BODY=$(echo "$RESP" | sed '$d' || true)

if [[ "$HTTP_STATUS" != "500" ]]; then
    error "Expected HTTP 500 for size mismatch, got $HTTP_STATUS. Body: $BODY"
    exit 3
fi

if ! echo "$BODY" | grep -q "size_mismatch"; then
    error "Expected 'size_mismatch' in response body, got: $BODY"
    exit 4
fi

success "size mismatch test passed for slug $SLUG"

exit 0

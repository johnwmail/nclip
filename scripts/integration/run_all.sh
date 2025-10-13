#!/bin/bash
set -euo pipefail
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$DIR/lib.sh"

SCRIPTS=(
    "$DIR/test_health.sh"
    "$DIR/test_paste.sh"
    "$DIR/test_burn.sh"
    "$DIR/test_extensions_text.sh"
    "$DIR/test_extensions_binary.sh"
    "$DIR/test_not_found.sh"
    "$DIR/test_xttl.sh"
    "$DIR/test_slug_collision.sh"
    "$DIR/test_expired.sh"
    "$DIR/test_buffer.sh"
    "$DIR/test_preview.sh"
    "$DIR/test_size_mismatch.sh"
)

failed=0
for s in "${SCRIPTS[@]}"; do
    echo
    log "Running $s"
    if ! bash "$s"; then
        warn "$s failed"
        failed=$((failed+1))
    else
        success "$s passed"
    fi
done

if [[ $failed -eq 0 ]]; then
    success "All integration scripts passed"
    exit 0
else
    error "$failed integration script(s) failed"
    exit 1
fi

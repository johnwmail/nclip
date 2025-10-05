#!/bin/bash
set -euo pipefail

# Shared helpers for integration tests

# Resolve this script dir
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Configuration (can be overridden by environment)
NCLIP_URL="${NCLIP_URL:-http://localhost:8080}"
SLUG_LENGTH="${SLUG_LENGTH:-5}"
TEST_TIMEOUT=30
RETRY_DELAY=2
MAX_RETRIES=15

TRASH_RECORD_FILE="/tmp/nclip_integration_slugs.txt"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YIGHL='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() { echo -e "${BLUE}[INFO]${NC} $*"; }
warn() { echo -e "${YIGHL}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; }
success() { echo -e "${GREEN}[SUCCESS]${NC} $*"; }

record_slug() {
    local slug="$1"
    if [[ -n "$slug" ]]; then
        echo "$slug" >> "$TRASH_RECORD_FILE" || true
    fi
}

cleanup_temp_files() {
    log "Cleaning up all test artifacts..."
    if [[ -f "$TRASH_RECORD_FILE" ]]; then
        log "Removing recorded data files listed in $TRASH_RECORD_FILE"
        while IFS= read -r slug; do
            if [[ -n "$slug" ]]; then
                rm -f "./data/${slug}" "./data/${slug}.json" || true
            fi
        done < "$TRASH_RECORD_FILE"
        rm -f "$TRASH_RECORD_FILE" || true
    else
        if [[ -d "./data" ]]; then
            find ./data -maxdepth 1 -type f -mmin -60 -print -delete || true
        fi
    fi
    rm -f /tmp/nclip_test_* /tmp/nclip_test_ext.* 2>/dev/null || true
    if [[ -d "./testdata" ]]; then
        rmdir ./testdata 2>/dev/null || true
    fi
    return 0
}

# Retry helper for POST requests (body-only). Usage: try_post VAR_NAME URL DATA
try_post() {
    local _varname="$1"; shift
    local _url="$1"; shift
    local _data="$1"; shift
    local _attempt=0
    local _max=3
    local _resp
    local auth_header=()
    if [[ -n "${NCLIP_TEST_API_KEY:-}" ]]; then
        auth_header=( -H "X-Api-Key: ${NCLIP_TEST_API_KEY}" )
    elif [[ -n "${NCLIP_TEST_API_KEY_BEARER:-}" ]]; then
        auth_header=( -H "Authorization: Bearer ${NCLIP_TEST_API_KEY_BEARER}" )
    fi

    while true; do
        _attempt=$((_attempt+1))
        _resp=$(curl -sS -w "\n%{http_code}" -X POST "${_url}" "${auth_header[@]}" -d "${_data}" 2>/dev/null || true)
        # split response and status
        local _status
        _status=$(echo "${_resp}" | tail -n1)
        local _body
        _body=$(echo "${_resp}" | sed '$d')
        if [[ "${_status}" =~ ^2[0-9][0-9]$ ]]; then
            printf -v "${_varname}" '%s' "${_body}"
            return 0
        fi
        if [[ $_attempt -ge $_max ]]; then
            printf -v "${_varname}" '%s' "${_body}"
            return 22
        fi
        warn "POST to ${_url} failed (status=${_status}). Retrying (${_attempt}/${_max})..."
        sleep 1
    done
}

# Initialize auth header args for generic curl calls
init_auth() {
    CURL_AUTH_ARGS=()
    if [[ -n "${NCLIP_TEST_API_KEY:-}" ]]; then
        CURL_AUTH_ARGS=( -H "X-Api-Key: ${NCLIP_TEST_API_KEY}" )
    elif [[ -n "${NCLIP_TEST_API_KEY_BEARER:-}" ]]; then
        CURL_AUTH_ARGS=( -H "Authorization: Bearer ${NCLIP_TEST_API_KEY_BEARER}" )
    fi
}

cget() { curl -sS "${CURL_AUTH_ARGS[@]}" "$@"; }
cget_f() { curl -f -sS "${CURL_AUTH_ARGS[@]}" "$@"; }
cpost() { curl -sS "${CURL_AUTH_ARGS[@]}" "$@"; }

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

# Run init on source
init_auth

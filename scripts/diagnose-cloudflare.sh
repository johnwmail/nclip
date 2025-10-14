#!/usr/bin/env bash
# Diagnostic script for Cloudflare + Lambda upload issues
# Usage: ./scripts/diagnose-cloudflare.sh https://demo.nclip.app

set -euo pipefail

URL="${1:-}"
if [[ -z "$URL" ]]; then
    echo "Usage: $0 <url>"
    echo "Example: $0 https://demo.nclip.app"
    exit 1
fi

echo "==================================="
echo "nclip Cloudflare Diagnostic Script"
echo "==================================="
echo ""
echo "Testing URL: $URL"
echo ""

# Test 1: Health check
echo "[TEST 1] Health Check"
echo "---------------------"
if curl -s -f "$URL/health" > /dev/null 2>&1; then
    echo "✓ Health endpoint accessible"
    curl -s "$URL/health" | jq '.' 2>/dev/null || curl -s "$URL/health"
else
    echo "✗ Health endpoint failed"
fi
echo ""

# Test 2: Check response headers
echo "[TEST 2] Response Headers"
echo "-------------------------"
curl -s -I "$URL/" | grep -E "server|cf-|cloudflare" || echo "No Cloudflare headers detected"
echo ""

# Test 3: Simple text upload
echo "[TEST 3] Simple Text Upload"
echo "---------------------------"
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$URL/" \
    -H "Content-Type: text/plain" \
    -H "User-Agent: nclip-diagnostic/1.0" \
    -d "diagnostic test content")

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
BODY=$(echo "$RESPONSE" | grep -v "HTTP_CODE:")

echo "HTTP Status: $HTTP_CODE"
echo "Response Body:"
echo "$BODY"

# Check if response is HTML (Cloudflare error)
if echo "$BODY" | grep -q "<!DOCTYPE\|<html"; then
    echo ""
    echo "⚠️  WARNING: Received HTML response (likely Cloudflare error page)"
    echo "    This indicates Cloudflare is blocking the request BEFORE it reaches Lambda"
    echo ""
    echo "Recommended Actions:"
    echo "  1. Check Cloudflare Firewall Events (Security > Events)"
    echo "  2. Look for blocked POST requests to /"
    echo "  3. Temporarily disable Bot Fight Mode"
    echo "  4. Create Page Rule to bypass security for API endpoints"
    echo "  5. Or use direct Lambda Function URL (bypass Cloudflare entirely)"
elif echo "$BODY" | grep -q '"url"\|"slug"'; then
    echo ""
    echo "✓ Upload successful! JSON response received"
    SLUG=$(echo "$BODY" | grep -o '"slug":"[^"]*"' | cut -d'"' -f4)
    if [[ -n "$SLUG" ]]; then
        echo "  Paste URL: $URL/$SLUG"
    fi
else
    echo ""
    echo "⚠️  Unexpected response format"
fi
echo ""

# Test 4: Binary upload (if text succeeded)
if [[ "$HTTP_CODE" == "200" ]]; then
    echo "[TEST 4] Binary File Upload"
    echo "---------------------------"
    
    # Create temporary binary file
    TMPFILE=$(mktemp)
    dd if=/dev/urandom bs=1024 count=10 of="$TMPFILE" 2>/dev/null
    
    RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$URL/" \
        -H "Content-Type: application/octet-stream" \
        -H "User-Agent: nclip-diagnostic/1.0" \
        --data-binary "@$TMPFILE")
    
    HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
    BODY=$(echo "$RESPONSE" | grep -v "HTTP_CODE:")
    
    rm -f "$TMPFILE"
    
    echo "HTTP Status: $HTTP_CODE"
    echo "Response Body:"
    echo "$BODY"
    
    if echo "$BODY" | grep -q "<!DOCTYPE\|<html"; then
        echo ""
        echo "⚠️  WARNING: Binary upload blocked by Cloudflare"
        echo "    Binary uploads may trigger WAF rules"
    elif echo "$BODY" | grep -q '"url"\|"slug"'; then
        echo ""
        echo "✓ Binary upload successful!"
    fi
    echo ""
fi

# Test 5: Large file upload
echo "[TEST 5] Large File Upload (1MB)"
echo "---------------------------------"
TMPFILE=$(mktemp)
dd if=/dev/zero bs=1024 count=1024 of="$TMPFILE" 2>/dev/null

RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$URL/" \
    -H "Content-Type: text/plain" \
    -H "User-Agent: nclip-diagnostic/1.0" \
    --data-binary "@$TMPFILE")

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
BODY=$(echo "$RESPONSE" | grep -v "HTTP_CODE:")

rm -f "$TMPFILE"

echo "HTTP Status: $HTTP_CODE"
if echo "$BODY" | grep -q "<!DOCTYPE\|<html"; then
    echo "⚠️  Large file upload blocked (likely Cloudflare payload limit)"
elif echo "$BODY" | grep -q "too large"; then
    echo "✓ Lambda received request and returned size limit error (expected)"
elif echo "$BODY" | grep -q '"url"\|"slug"'; then
    echo "✓ Large file upload successful"
fi
echo ""

# Summary
echo "==================================="
echo "Diagnostic Summary"
echo "==================================="
echo ""
echo "If you see HTML responses with '<!DOCTYPE html>':"
echo "  → Cloudflare is blocking requests BEFORE they reach Lambda"
echo "  → Check Cloudflare Security Events"
echo "  → Consider bypassing Cloudflare for /api/* endpoints"
echo ""
echo "If you see JSON responses:"
echo "  → Lambda is working correctly"
echo "  → Issue may be with specific file types or sizes"
echo ""
echo "For detailed troubleshooting, see:"
echo "  Documents/LAMBDA.md (Section: Troubleshooting > Cloudflare Issues)"
echo ""

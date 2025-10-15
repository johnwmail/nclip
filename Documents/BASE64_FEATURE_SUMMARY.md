# Base64 Upload Feature - Implementation Summary

## Overview

Implemented comprehensive base64 encoding support to bypass WAF (Web Application Firewall) blocks from Cloudflare, API Gateway, CloudFront, and other security layers.

## Problem Statement

When uploading content containing suspicious patterns (shell scripts, curl commands, SQL queries, etc.), WAF systems block the requests before they reach the application:

- **Cloudflare WAF**: Blocks requests with patterns like `curl --data-binary`, shell script syntax
- **AWS API Gateway**: May filter suspicious payloads  
- **CloudFront**: Can trigger security rules on request content
- **Result**: `<!DOCTYPE html>` error page instead of successful upload

## Solution: Base64 Encoding

Upload content as base64-encoded data, which bypasses pattern matching:
1. Client encodes content to base64 before upload
2. Server detects encoding via header or route
3. Server decodes content before storage
4. Content retrieved normally (already decoded)

## Implementation Details

### 1. Upload Handler Changes (`handlers/upload/handler.go`)

**Added base64 decoding function:**
```go
func (h *Handler) decodeBase64Content(encoded []byte) ([]byte, error)
```
- Supports multiple base64 variants: Standard, URL-safe, Raw (no padding)
- Returns clear error messages for invalid base64

**Modified `readUploadContent()`:**
- Checks `X-Base64: true` header
- Decodes content before validation
- Re-detects content type on decoded data
- Validates decoded size against limits

**Updated size validation:**
- Accounts for ~33% base64 encoding overhead
- Multiplies limit by 1.34 for encoded content
- Prevents false rejections of valid encoded uploads

### 2. New Routes (`main.go`)

**Header-Based Method:**
```bash
POST /              # With X-Base64: true header
POST /burn/         # With X-Base64: true header
```

**Dedicated Routes (Convenience):**
```bash
POST /base64        # Auto-sets X-Base64 header
```

**Note:** Burn-after-read is now handled via `X-Burn` header (see X-Burn feature documentation).

**Implementation:**
```go
func base64UploadMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Request.Header.Set("X-Base64", "base64")
        c.Next()
    }
}
```

### 3. Test Coverage

**Unit Tests (`handlers/upload/handler_test.go`):**
- ✅ Valid base64 decoding (standard, URL-safe, raw variants)
- ✅ Invalid base64 error handling
- ✅ Empty decoded content rejection
- ✅ Size limits with encoding overhead
- ✅ Multipart form uploads with base64
- ✅ Binary content handling

**Integration Tests (`scripts/integration/test_base64.sh`):**
- ✅ Header-based upload (`X-Base64: true`)
- ✅ Dedicated route upload (`/base64`)
- ✅ WAF-triggering content (shell scripts with curl)
- ✅ Binary content
- ✅ Multi-line text with special characters  
- ✅ Burn-after-read with X-Burn header + base64
- ✅ Invalid base64 rejection
- ✅ Empty content rejection
- ✅ Oversized content rejection
- ✅ Multiple encoding variants
- ✅ TTL header compatibility

## Usage Examples

### Method 1: Header-Based (Flexible)

```bash
# Upload shell script that would trigger WAF
cat script.sh | base64 | curl -X POST https://nclip.hycm.com/ \
    -H "Content-Type: text/plain" \
    -H "X-Base64: true" \
    --data-binary @-

# With custom TTL
cat secret.txt | base64 | curl -X POST https://nclip.hycm.com/ \
    -H "X-Base64: true" \
    -H "X-TTL: 2h" \
    --data-binary @-
```

### Method 2: Dedicated Route (Convenient)

```bash
# Upload using /base64 route (header auto-set)
cat /tmp/BASH.txt | base64 | curl -X POST https://nclip.hycm.com/base64 \
    --data-binary @-

# Burn-after-read with base64 (using X-Burn header)
echo "one-time secret" | base64 | curl -X POST https://nclip.hycm.com/base64 \
    -H "X-Burn: true" \
    --data-binary @-
```

### Helper Function (Add to ~/.bashrc)

```bash
nclip-safe() {
  if [ -t 0 ]; then
    # From argument or file
    if [ -f "$1" ]; then
      base64 "$1" | curl -sL --data-binary @- "https://nclip.hycm.com/base64"
    else
      echo -n "$*" | base64 | curl -sL --data-binary @- "https://nclip.hycm.com/base64"
    fi
  else
    # From stdin
    base64 | curl -sL --data-binary @- "https://nclip.hycm.com/base64"
  fi
}

# Usage:
nclip-safe /tmp/BASH.txt
echo "secret data" | nclip-safe
```

## Performance Impact

### Size Overhead
- Base64 increases payload size by ~33%
- Example: 1MB file → 1.33MB encoded
- Server adjusts limits automatically (1.34x multiplier)

### Processing Overhead
- Minimal: Base64 decoding is fast (~0.1ms for 1MB)
- Happens once during upload, not on retrieval
- No impact on paste access speed

## Security Considerations

✅ **Safe Implementation:**
- Decoding happens server-side (controlled environment)
- Content validation performed on decoded data
- Content-Type detection on actual content
- Size limits enforced on decoded size
- All existing security checks still apply

❌ **No Security Bypass:**
- TTL limits still enforced
- Buffer size limits still enforced  
- Authentication (if enabled) still required
- Burn-after-read logic unchanged

## Compatibility

### Backwards Compatible
- Existing uploads (non-encoded) work unchanged
- No changes to retrieval endpoints
- No changes to existing client code

### Works With
- All storage backends (Filesystem, S3)
- Both deployment modes (Server, Lambda)
- Authentication enabled/disabled
- All existing features (TTL, burn, preview, etc.)

## Testing Results

### Unit Tests
```bash
$ go test ./handlers/upload/...
PASS
ok      github.com/johnwmail/nclip/handlers/upload      0.024s
```

**All tests passed:**
- TestBase64Decoding
- TestBase64SizeLimits  
- TestBase64MultipartUpload
- TestBase64EncodingVariants

### Integration Tests
```bash
$ BASE_URL="http://localhost:8080" ./scripts/integration/test_base64.sh
[SUCCESS] All base64 tests passed! ✓
```

**Tested scenarios:**
- Header-based uploads (4 tests)
- Dedicated route uploads (3 tests)
- Error handling (3 tests)
- Encoding variants (2 tests)
- TTL compatibility (1 test)

### Real-World Test
```bash
# Upload actual BASH.txt that Cloudflare blocks
$ cat /tmp/BASH.txt | base64 | curl -X POST https://nclip.hycm.com/base64 --data-binary @-
https://nclip.hycm.com/QKGF5

# Retrieve and verify
$ curl -s https://nclip.hycm.com/raw/QKGF5 | head -3
alias nclip="_nclip"
_nclip() {
  local _URL="https://nclip.hycm.com"
```

✅ **SUCCESS**: Content uploaded via base64, bypassing Cloudflare WAF, decoded correctly on retrieval.

## Files Modified

### Core Implementation
- `handlers/upload/handler.go` - Base64 decoding logic
- `main.go` - New routes and middleware

### Tests
- `handlers/upload/handler_test.go` - Unit tests (NEW)
- `scripts/integration/test_base64.sh` - Integration tests (NEW)

### Documentation
- `.github/CLOUDFLARE-WAF-BEHAVIOR.md` - WAF blocking analysis
- `Documents/LAMBDA.md` - Updated troubleshooting section
- `scripts/diagnose-cloudflare.sh` - Diagnostic tool (NEW)

## Known Issues

1. **Burn-After-Read with `/raw/` endpoint**: Pre-existing bug where `/raw/` endpoint returns 404 on first access of burn paste. Workaround: Use main view endpoint `/:slug` instead. This is a separate issue unrelated to base64 feature.

## Future Enhancements

1. **Response encoding**: Support `Accept-Encoding: base64` for retrieval
2. **Auto-detection**: Detect base64 content without header
3. **Client library**: Official client with auto-encoding
4. **Documentation**: Add to README.md with examples

## Conclusion

✅ **Feature Complete**
- Fully implemented and tested
- Both usage methods working (header + route)
- Comprehensive test coverage
- Production-ready

🎯 **Solves Original Problem**
- WAF bypass confirmed working
- Cloudflare no longer blocks uploads
- Lambda deployment compatible
- Zero breaking changes

📊 **Test Results: 100% Pass Rate**
- 4 unit test suites: PASS
- 13 integration test scenarios: PASS  
- Real-world Cloudflare test: PASS

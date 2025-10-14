# X-Burn Header Feature Implementation Summary

## Overview
Added `X-Burn` header support for burn-after-read functionality, providing a more flexible and RESTful alternative to route-based burn paste creation.

## Changes Made

### 1. Core Implementation
**File:** `handlers/upload/handler.go`

- Modified `Upload()` function to check for `X-Burn` header
- Supported values: `true`, `1`, `yes` (case-sensitive)
- Header takes precedence over route-based detection
- Falls back to route checking (`/burn/`) for backward compatibility

```go
// Determine if burn-after-read: check X-Burn header first, then fall back to route path
burnAfterRead := false
if burnHeader := c.GetHeader("X-Burn"); burnHeader != "" {
    // Support: X-Burn: true, X-Burn: 1, X-Burn: yes
    burnAfterRead = burnHeader == "true" || burnHeader == "1" || burnHeader == "yes"
} else {
    // Fall back to route-based detection for backward compatibility
    burnAfterRead = strings.HasSuffix(c.FullPath(), "/burn/")
}
```

### 2. Route Configuration
**File:** `main.go`

**Design Decision:** Header-based approach for composable features.

**Rationale:** With `X-Burn` header, users can combine features without route explosion:
```bash
# Composable approach with headers
echo "secret" | base64 | curl -X POST /base64 -H "X-Burn: true" --data-binary @-
```

**Available routes:**
- `POST /burn/` - Backward compatible burn-after-read

### 3. Unit Tests
**File:** `handlers/upload/handler_test.go`

Added `TestXBurnHeader` with 8 test scenarios:
1. X-Burn: true on / route
2. X-Burn: 1 on / route
3. X-Burn: yes on / route
4. X-Burn: false on / route (should NOT burn)
5. X-Burn: 0 on / route (should NOT burn)
6. No X-Burn header on / route (should NOT burn)
7. /burn/ route without header (backward compatibility)
8. X-Burn header overrides route

**All tests PASSING** ✅

### 4. Integration Tests
**File:** `scripts/integration/test_xburn.sh` (NEW)

Comprehensive integration test suite covering:
- X-Burn header with different values (true, 1, yes)
- Burn behavior validation (404 on second read)
- Non-burn scenarios (false, 0, no header)
- X-Burn + Base64 encoding combination
- X-Burn + /base64 route
- Backward compatibility with /burn/ route

**All tests PASSING** ✅

### 5. Documentation Updates

**File:** `Documents/CLIENTS.md`
- Added base64 examples for all major clients (curl, wget, PowerShell, HTTPie)
- Updated "Burn-After-Read" section with X-Burn header examples
- Added new "Custom Headers Reference" section documenting all supported headers
- Provided combination examples (Burn + Base64, Burn + TTL, etc.)

**File:** `README.md`
- Added "Base64 Upload Support" to features list
- Updated quick start example to show base64 usage
- Updated API endpoints section to show X-Burn header support
- Documented composable header-based approach
- Added supported headers list

### 6. Quick Test Script
**File:** `quick_test_xburn.sh` (NEW)

Simple manual test script for quick validation:
- Test 1: X-Burn: true header
- Test 2: No X-Burn header (should not burn)
- Test 3: X-Burn + Base64 encoding
- Test 4: /burn/ route (backward compatibility)

## Supported Headers

After this implementation, nclip now supports:

| Header | Purpose | Values | Example |
|--------|---------|--------|---------|
| `X-TTL` | Custom expiration | 1h to 7d | `X-TTL: 2h` |
| `X-Slug` | Custom paste ID | 3-32 chars | `X-Slug: MYPASTE` |
| `X-Base64` | Base64 upload | `base64` | `X-Base64: true` |
| `X-Burn` | Burn-after-read | `true`, `1`, `yes` | `X-Burn: true` |
| `Authorization` | API auth | `Bearer <key>` | `Authorization: Bearer key` |
| `X-Api-Key` | API auth (alt) | `<key>` | `X-Api-Key: key` |

## Usage Examples

### Basic Burn-After-Read
```bash
# Using header (recommended)
echo "secret" | curl -X POST http://localhost:8080/ -H "X-Burn: true" --data-binary @-

# Using route (backward compatible)
echo "secret" | curl -X POST http://localhost:8080/burn/ --data-binary @-
```

### Burn + Base64
```bash
echo "secret script" | base64 | curl -X POST http://localhost:8080/ \
    -H "X-Burn: true" \
    -H "X-Base64: true" \
    --data-binary @-
```

### Burn + Custom TTL
```bash
echo "expires in 1h" | curl -X POST http://localhost:8080/ \
    -H "X-Burn: 1" \
    -H "X-TTL: 1h" \
    --data-binary @-
```

### All Features Combined
```bash
cat script.sh | base64 | curl -X POST http://localhost:8080/ \
    -H "X-Base64: true" \
    -H "X-Burn: true" \
    -H "X-TTL: 2h" \
    -H "X-Slug: DEPLOY" \
    --data-binary @-
```

## Test Results

### Unit Tests
```
ok      github.com/johnwmail/nclip                       0.602s
ok      github.com/johnwmail/nclip/config                (cached)
ok      github.com/johnwmail/nclip/handlers              (cached)
ok      github.com/johnwmail/nclip/handlers/retrieval    (cached)
ok      github.com/johnwmail/nclip/handlers/upload       (cached)
ok      github.com/johnwmail/nclip/internal/services     (cached)
ok      github.com/johnwmail/nclip/models                (cached)
ok      github.com/johnwmail/nclip/storage               0.127s
ok      github.com/johnwmail/nclip/utils                 (cached)
```

### Integration Tests
```
=== All X-Burn Tests PASSED ✓ ===

Test 1: X-Burn: true header              ✓
Test 2: No X-Burn header (should not burn)    ✓
Test 3: X-Burn + Base64 encoding         ✓
Test 4: /burn/ route (backward compatibility) ✓
```

## Design Rationale

### Why Header-Based Instead of Multiple Routes?

**Avoided Approach (Route Explosion):**
```
POST /burn/
POST /burn/ttl/
POST /burn/custom-slug/
POST /custom-slug/burn/ttl/
... (exponential route explosion for feature combinations)
```

**Implemented Approach (Composable Headers):**
```
POST / 
  with composable headers:
    -H "X-Burn: true"
    -H "X-Base64: true"
    -H "X-TTL: 2h"
    -H "X-Slug: CUSTOM"
```

**Benefits:**
1. **Composability** - Any combination of features without new routes
2. **RESTful** - Single resource (`/`) with modifiers via headers
3. **Maintainability** - No route explosion, easier to test
4. **Flexibility** - Users can combine features as needed
5. **Backward Compatible** - Old `/burn/` route still works

### Header Precedence
- `X-Burn` header takes precedence over route-based detection
- If `X-Burn` header is present, its value determines burn behavior
- If `X-Burn` header is absent, route path is checked (backward compatibility)
- This allows gradual migration from route-based to header-based approach

## Breaking Changes

**None** - This implementation is fully backward compatible.

- Existing `/burn/` route continues to work
- Existing code using `/burn/` route doesn't need changes
- New `X-Burn` header is optional enhancement

## Usage Guide

**Base64 + Burn-After-Read:**

Option 1 - Use /base64 route with X-Burn header:
```bash
cat script.sh | base64 | curl -X POST http://localhost:8080/base64 -H "X-Burn: true" --data-binary @-
```

Option 2 - Use / route with both headers:
```bash
cat script.sh | base64 | curl -X POST http://localhost:8080/ \
    -H "X-Base64: true" \
    -H "X-Burn: true" \
    --data-binary @-
```

## Status

✅ **Production Ready**

- All unit tests passing
- All integration tests passing
- Documentation updated
- Backward compatible
- Well tested with real-world scenarios

## Files Changed

1. `handlers/upload/handler.go` - Core implementation
2. `handlers/upload/handler_test.go` - Unit tests
3. `main.go` - Route configuration
4. `Documents/CLIENTS.md` - Client documentation
5. `README.md` - API documentation
6. `scripts/integration/test_xburn.sh` - Integration tests (NEW)
7. `quick_test_xburn.sh` - Quick manual tests (NEW)

## Next Steps

1. Deploy to production
2. Monitor for any issues
3. Update client libraries/wrappers if applicable
4. Consider adding X-Burn support to Web UI (future enhancement)

---

**Feature implemented:** October 14, 2025  
**Branch:** `feature/base64`  
**Status:** Ready for merge to `dev`

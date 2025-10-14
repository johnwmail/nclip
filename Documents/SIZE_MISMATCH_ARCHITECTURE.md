# Size Mismatch Detection Architecture

This document explains how nclip detects and handles size mismatches between stored metadata and actual file content.

## Overview

Size mismatch detection prevents serving corrupted or tampered content by verifying that the actual file size matches the metadata before streaming content to clients.

## Architecture: Single Point of Verification

### Design Principle
**Verify once at the entry point, trust in all downstream handlers.**

```
┌─────────────────────────────────────────────────────────────┐
│                       View() [GET /:slug]                    │
│                                                              │
│  1. Validate slug format                                    │
│  2. Get paste metadata                                      │
│  3. ✓ EARLY SIZE CHECK (StatContent)  ← SINGLE CHECK POINT │
│  4. Increment read count                                    │
│  5. Route to appropriate handler (no re-checking)           │
└─────────────────────────────────────────────────────────────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
           ▼               ▼               ▼
    ┌──────────┐   ┌──────────────┐  ┌──────────┐
    │ viewCLI  │   │viewBrowserBurn│  │viewBrowser│
    │ (no check)│   │  (no check)   │  │ (no check)│
    └──────────┘   └──────────────┘  └──────────┘
         │                │               │
         └────────────────┴───────────────┘
                        │
                   Stream Content
```

## Call Chain Analysis

### Who Calls What?

```go
View() → Entry point, called by Gin router for GET /:slug
  ├─ Performs StatContent() size verification ONCE
  │
  ├─ if paste.BurnAfterRead:
  │    ├─ if isCli() → viewCLI()       // No size check
  │    └─ else → viewBrowserBurn()     // No size check
  │
  └─ else (non-burn):
       ├─ if isCli() → viewCLI()       // No size check
       └─ else → viewBrowser()         // No size check

Raw() → Entry point, called by Gin router for GET /raw/:slug
  └─ Performs StatContent() size verification independently
```

## Implementation Details

### Size Check Location: View() Function

```go
func (h *Handler) View(c *gin.Context) {
    // 1. Get metadata
    paste, err := h.service.GetPaste(slug)
    
    // 2. SINGLE SIZE CHECK - Early verification using StatContent
    if exists, actualSize, serr := h.store.StatContent(slug); serr == nil && exists {
        if actualSize != paste.Size {
            // Fail immediately with 500 + "size_mismatch" error
            return
        }
    }
    
    // 3. Route to handlers - all downstream functions trust this check
    if paste.BurnAfterRead {
        if h.isCli(c) {
            h.viewCLI(c, slug, paste)  // ✓ No redundant check
        } else {
            h.viewBrowserBurn(c, slug, paste)  // ✓ No redundant check
        }
    } else {
        if h.isCli(c) {
            h.viewCLI(c, slug, paste)  // ✓ No redundant check
        } else {
            h.viewBrowser(c, slug, paste)  // ✓ No redundant check
        }
    }
}
```

### Why StatContent()?

`StatContent()` is a lightweight operation that checks file size without reading content:
- **Filesystem backend**: Uses `os.Stat()` (system call, no I/O)
- **S3 backend**: Uses `HeadObject()` (metadata only, no data transfer)

This is **much cheaper** than reading the full content just to verify size.

## Removed Redundancy

### Before Simplification (❌ 3 checks)

```
View()            → StatContent() check ✓
├─ viewCLI()      → StatContent() check ✗ REDUNDANT
├─ viewBrowserBurn() → len(content) check ✗ REDUNDANT
└─ viewBrowser()  → No check
```

### After Simplification (✅ 1 check)

```
View()            → StatContent() check ✓ ONLY CHECK
├─ viewCLI()      → No check (trusts View)
├─ viewBrowserBurn() → No check (trusts View)
└─ viewBrowser()  → No check (never needed one)
```

## Benefits of Single-Point Verification

1. **Performance**: Only one stat operation per request
2. **Consistency**: All paths get identical size verification
3. **Maintainability**: Single source of truth for size checking logic
4. **Simplicity**: Helper functions focus on their core responsibility
5. **DRY Principle**: Don't Repeat Yourself - no duplicate checks

## Error Response Format

When size mismatch is detected:

**CLI/API clients:**
```json
{
  "error": "size_mismatch"
}
```
HTTP Status: `500 Internal Server Error`

**Browser clients:**
```html
<view.html with Error="Size mismatch">
```
HTTP Status: `500 Internal Server Error`

## Related Functions

### Helper Functions (No Size Checks)

- `viewCLI()` - Streams content to CLI clients (curl/wget)
- `viewBrowserBurn()` - Renders burn-after-read for browsers
- `viewBrowser()` - Renders normal content for browsers
- `loadFullContent()` - Loads complete content for small files
- `loadPreviewContent()` - Loads preview for large files

### Entry Points (Size Checks)

- `View()` - Handles GET /:slug (✓ has size check)
- `Raw()` - Handles GET /raw/:slug (✓ has independent size check)

## Testing

All size mismatch scenarios are tested:

```go
// handlers/retrieval/handler_test.go
func TestView_SizeMismatch_CLI(t *testing.T)
func TestRaw_SizeMismatch_NonBurn(t *testing.T)

// scripts/integration/test_size_mismatch.sh
- Integration test that truncates files to simulate corruption
- Verifies both View and Raw endpoints reject mismatched content
```

## Migration Notes

**Previous behavior:** Size verification occurred at multiple points (View, viewCLI, viewBrowserBurn)

**Current behavior:** Size verification occurs only once in View() before routing to handlers

**Benefit:** Eliminated ~30 lines of redundant code while maintaining identical safety guarantees

## See Also

- `handlers/retrieval/handler.go` - Implementation
- `storage/interface.go` - StatContent interface definition
- `storage/filesystem.go` - Filesystem StatContent implementation
- `storage/s3.go` - S3 StatContent implementation

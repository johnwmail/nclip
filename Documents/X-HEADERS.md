# X-HEADERS Reference for nclip

This document centralizes the supported HTTP `X-` headers used by nclip, their purpose, accepted values, examples, and notes about route precedence and middleware behavior.

Status: Consolidated from `BASE64_FEATURE_SUMMARY.md` and `XBURN_FEATURE_SUMMARY.md`.

## Quick summary

- X-Base64 — instructs the server that the request body is base64-encoded and should be decoded before storage.
- X-Burn — marks a paste to burn-after-read (delete after the first successful retrieval).
- X-TTL — custom time-to-live for a paste (duration string between 1h and 7d).
- X-Slug — custom paste identifier (validated, see `utils.IsValidSlug`).
- Authorization / X-Api-Key — API auth headers (when `NCLIP_UPLOAD_AUTH` is enabled).

All headers are optional. Many features are composable using headers (for example: `X-Base64` + `X-Burn` + `X-TTL`).

---

## Behavior and precedence notes

- Route-based shortcuts exist for convenience, but they may set headers internally:
  - `POST /base64` uses a middleware that sets `X-Base64: true` for the request (see `main.go:base64UploadMiddleware`). That currently replaces any client-provided `X-Base64` header value.
  - `POST /burn/` is a dedicated route that creates a burn-after-read paste; the handler used for this route returns a paste with BurnAfterRead set to true regardless of incoming `X-Burn` header.

- Header semantics (server-side): nclip uses presence-based header semantics with explicit opt-out tokens. If a header is present, it is treated as enabled unless the value is an explicit disabling token: `0`, `false`, or `no` (case-insensitive). For example:
  - `X-Base64:` (header present with empty value) -> treated as enabled
  - `X-Base64: false` -> treated as disabled

  This logic is implemented centrally in `handlers/upload/headerEnabled()`.

- For the canonical upload routes:
  - `POST /` — headers (e.g. `X-Base64`, `X-Burn`, `X-TTL`, `X-Slug`) are honored according to the presence/disable semantics.
  - `POST /base64` — convenience route that sets `X-Base64: true` via middleware before the handler runs; this currently overwrites client header values (so `X-Base64: false` sent by a client will be replaced with `true`).
  - `POST /burn/` — dedicated route that creates a burn paste; to opt out of burn on `POST /burn/` you must not use this route (POST to `/` and omit/disable `X-Burn`).

If you prefer route defaults that are overridable by client headers, consider changing the middleware to only set a header when it is not already present (see `main.go:base64UploadMiddleware`).

---

## X-Base64

Purpose: indicate the request body is base64-encoded and should be decoded before validation and storage. Useful to bypass WAFs and proxies that inspect request content.

Accepted values and semantics:
- Presence-enabled by default: if the header key `X-Base64` is present, the server treats the upload as base64-encoded unless the value equals `0`, `false`, or `no` (case-insensitive).
- Typical values to explicitly enable: `true`, `1`, `yes` (any non-disabling value is treated as enabled).

Implementation notes:
- Decoding supports multiple variants: standard base64, URL-safe, raw (no padding) variants (`base64.StdEncoding`, `base64.URLEncoding`, `base64.RawStdEncoding`, `base64.RawURLEncoding`). See `handlers/upload/handler.go:decodeBase64Content()`.
- Server re-detects content type after decoding and validates decoded size against the configured limit (the configured buffer limit applies to decoded bytes).
- When base64 is enabled the code accounts for encoding overhead (~33%) by multiplying the configured buffer size by 1.34 for upload-limit checks.

Examples:

Header-based method (flexible):

```bash
cat script.sh | base64 | curl -X POST https://example.com/ \
  -H "Content-Type: text/plain" \
  -H "X-Base64: true" \
  --data-binary @-
```

Dedicated route (convenient):

```bash
# POST /base64 sets X-Base64 internally via middleware
cat script.sh | base64 | curl -X POST https://example.com/base64 --data-binary @-
```

Edge cases:
- Invalid base64 payloads are rejected with an error message.
- Empty decoded content is rejected.
- Oversized decoded content is rejected (limit enforced on decoded length).

---

## X-Burn

Purpose: mark a paste to be deleted after the first successful retrieval (burn-after-read).

Accepted values and semantics:
- Presence-enabled: header presence enables burn unless the value is `0`, `false`, or `no` (case-insensitive).
- Explicitly enabling values: `true`, `1`, `yes` (or any non-disabling value).

Behavior:
- When `X-Burn` is present and enabled on `POST /`, the created paste will be marked as burn-after-read.
- `POST /burn/` remains a backward-compatible route that creates a burn paste regardless of header value; to avoid burn, post to `/` and control via header.

Examples:

```bash
# Using header on / route
echo "one-time" | curl -X POST https://example.com/ -H "X-Burn: true" --data-binary @-

# /burn/ route (convenience) — creates a burn paste regardless
echo "one-time" | curl -X POST https://example.com/burn/ --data-binary @-
```

Test notes:
- Integration tests include cases for `X-Burn` enabled/disabled and combinations with `X-Base64`.

---

## X-TTL

Purpose: set a custom lifetime for the paste.

Accepted values:
- A Go `time.Duration` string between `1h` and `7d` (e.g., `2h`, `24h`, `72h`).

Behavior:
- If `X-TTL` is present, the value is parsed; invalid values or values outside the allowed range cause a 400 error.
- If absent, the server uses the configured default TTL.

Example:
```bash
echo "short-lived" | curl -X POST https://example.com/ -H "X-TTL: 2h" --data-binary @-
```

---

## X-Slug

Purpose: request a custom paste ID/slug.

Accepted values:
- Validated by `utils.IsValidSlug` (alphanumeric, length constraints, etc.).
- If invalid, server returns 400.

Example:

```bash
echo "hello" | curl -X POST https://example.com/ -H "X-Slug: MYPASTE" --data-binary @-
```

---

## Authorization / X-Api-Key

Purpose: when upload authentication is enabled (`NCLIP_UPLOAD_AUTH`), clients supply credentials.

Accepted methods:
- `Authorization: Bearer <key>`
- `X-Api-Key: <key>`

Behavior:
- If `UploadAuth` is enabled, the API key middleware enforces presence of a configured key.

---

## Examples & Combined Usage

Create a base64-encoded, burn-after-read paste with a custom TTL and slug:

```bash
cat secret.sh | base64 | curl -X POST https://example.com/ \
  -H "X-Base64: true" \
  -H "X-Burn: true" \
  -H "X-TTL: 2h" \
  -H "X-Slug: DEPLOY" \
  --data-binary @-
```

Use `/base64` convenience route (note: this route sets `X-Base64: true` internally and may overwrite client header):

```bash
cat secret.sh | base64 | curl -X POST https://example.com/base64 --data-binary @-
```

To explicitly opt out of base64 when a route sets it by default (current behavior):
- Do not use the convenience route; instead `POST /` and send `X-Base64: false`.

---

## Implementation pointers (code locations)

- `handlers/upload/handler.go` — `readUploadContent`, `decodeBase64Content`, `readMultipartUpload`, `readDirectUpload`, and `headerEnabled` implementation.
- `main.go` — `base64UploadMiddleware` (sets `X-Base64` on `/base64` route), route registration.
- `scripts/integration/` — integration tests that exercise header behaviors: `test_xbase64.sh`, `test_xburn.sh`, etc.

---

## Migration notes

This file consolidates the previous `BASE64_FEATURE_SUMMARY.md` and `XBURN_FEATURE_SUMMARY.md` content. The old files remain in the repository for history; this file should be treated as the single source of truth for header behavior going forward.

---

## Changelog

- Consolidated base64 + burn header docs into `Documents/X-HEADERS.md` — 2025-10-15
- Documented presence-vs-value semantics and middleware precedence.

---

If you'd like I can also update `README.md` and `Documents/CLIENTS.md` to reference this file (I have that as a follow-up todo).
## nclip Client Usage Examples

This guide explains how to interact with nclip using various clients, including advanced features via custom HTTP headers like `X-TTL` and `X-SLUG`.

---

### 1. Curl (Linux/macOS/Windows)

**Upload text:**
```bash
echo "Hello World!" | curl -sL --data-binary @- http://localhost:8080
```


**Upload file (raw):**
```bash
curl -sL --data-binary @myfile.txt http://localhost:8080
```

**Upload file (multipart/form-data):**
```bash
curl -sL -F "file=@myfile.txt" http://localhost:8080
```

You can use either `--data-binary` for raw uploads or `-F/--form` for multipart uploads. Both are supported.

**Set custom TTL (expiry):**
```bash
echo "Expiring soon" | curl -sL --data-binary @- -H "X-TTL: 2h" http://localhost:8080
```

**Set custom slug/ID:**
```bash
echo "My custom slug" | curl -sL --data-binary @- -H "X-SLUG: MYPASTE" http://localhost:8080
```

**Upload with Base64 encoding (bypass WAF/security filters):**

Method 1 - Using dedicated route:
```bash
echo "Content with special chars" | base64 | curl -sL --data-binary @- http://localhost:8080/base64
```

Method 2 - Using header:
```bash
echo "Content with special chars" | base64 | curl -sL --data-binary @- -H "X-Base64: true" http://localhost:8080
```

Upload shell script (bypass WAF):
```bash
cat script.sh | base64 | curl -sL --data-binary @- http://localhost:8080/base64
```

Base64 + Burn-after-read (using X-Burn header):
```bash
echo "secret" | base64 | curl -sL --data-binary @- -H "X-Burn: true" http://localhost:8080/base64
```

Base64 + Custom TTL:
```bash
echo "expires soon" | base64 | curl -sL --data-binary @- -H "X-TTL: 2h" http://localhost:8080/base64
```

> **Note**: The server automatically decodes base64 content before storage. Retrieved content is returned in its original (decoded) form. This feature is useful for bypassing WAF/security filters that block certain patterns in plain text.

### API Key / Upload Auth Examples

If `NCLIP_UPLOAD_AUTH=true` is set on the server, upload endpoints require an API key. You can provide the key either via the `Authorization: Bearer <key>` header or the `X-Api-Key: <key>` header.

```bash
# Using Authorization: Bearer header
echo "protected content" | curl -sL --data-binary @- \
  -H "Authorization: Bearer my-secret-key" \
  http://localhost:8080

# Using X-Api-Key header
echo "protected content" | curl -sL --data-binary @- \
  -H "X-Api-Key: my-secret-key" \
  http://localhost:8080
```

For multipart/form uploads, include the header the same way:

```bash
curl -sL -F "file=@myfile.txt" -H "X-Api-Key: my-secret-key" http://localhost:8080
```

Web UI note: when upload auth is enabled the web UI includes an "API Key" input field. Paste your key there before uploading. Browsers will not automatically attach API keys for you.

---

### 2. Wget (Linux/macOS/Windows)

**Upload text:**
```bash
echo "Hello World!" | wget --method=POST --body-data="$(cat)" http://localhost:8080 -O -
```

**Upload file:**
```bash
wget --method=POST --body-file=myfile.txt http://localhost:8080 -O -
```

**Set custom TTL (expiry):**
```bash
echo "Expiring soon" | wget --method=POST --header="X-TTL: 2h" --body-data="$(cat)" http://localhost:8080 -O -
```

**Set custom slug/ID:**
```bash
echo "My custom slug" | wget --method=POST --header="X-SLUG: MYPASTE" --body-data="$(cat)" http://localhost:8080 -O -
```

**Upload with Base64 encoding:**
```bash
echo "Content with special chars" | base64 | wget --method=POST --body-data="$(cat)" http://localhost:8080/base64 -O -
```

---

### 3. PowerShell (Windows)

**Upload text:**
```powershell
Invoke-WebRequest -Uri http://localhost:8080 -Method POST -Body "Hello from PowerShell!" -UseBasicParsing
```

**Upload file:**
```powershell
Invoke-WebRequest -Uri http://localhost:8080 -Method POST -InFile "C:\path\to\file.txt" -UseBasicParsing
```

**Set custom TTL (expiry):**
```powershell
Invoke-WebRequest -Uri http://localhost:8080 -Method POST -Body "Expiring soon" -Headers @{"X-TTL"="2h"} -UseBasicParsing
```

**Set custom slug/ID:**
```powershell
Invoke-WebRequest -Uri http://localhost:8080 -Method POST -Body "My custom slug" -Headers @{"X-SLUG"="MYPASTE"} -UseBasicParsing
```

**Upload with Base64 encoding:**
```powershell
$content = "Content with special chars"
$base64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($content))
Invoke-WebRequest -Uri http://localhost:8080/base64 -Method POST -Body $base64 -UseBasicParsing
```

---

### 4. HTTPie (Linux/macOS/Windows)

**Upload text:**
```bash
echo "Hello World!" | http POST http://localhost:8080
```

**Upload file:**
```bash
http POST http://localhost:8080 < myfile.txt
```

**Set custom TTL (expiry):**
```bash
echo "Expiring soon" | http POST http://localhost:8080 X-TTL:2h
```

**Set custom slug/ID:**
```bash
echo "My custom slug" | http POST http://localhost:8080 X-SLUG:MYPASTE
```

**Upload with Base64 encoding:**
```bash
echo "Content with special chars" | base64 | http POST http://localhost:8080/base64
```

---

### 5. Bash Aliases (Linux/macOS)

You may find these bash aliases useful for working with nclip:

```bash
alias nclip="_nclip"
_nclip() {
  local _URL="https://demo.nclip.app"
  if [ -t 0 ]; then
    if [ $# -eq 1 ] && [ -f "$1" ]; then
      curl --data-binary @"$1" "$_URL"
    elif [ $# -eq 1 ] && [ ${#1} -eq 5 ] && [[ "$1" =~ ^[A-HJ-NP-Z2-9]{5}$ ]]; then
      curl -sL "$_URL/$1"
    else
      echo -en "$*" | curl --data-binary @- "$_URL"
    fi
  else
    cat | curl --data-binary @- "$_URL"
  fi
}
```

**Usage examples:**
```bash
# Upload text
nclip "Hello World!"

# Upload file
nclip myfile.txt

# Upload from stdin
echo "Hello from stdin" | nclip
```

---

### 6. Advanced: Burn-After-Read

Burn-after-read pastes self-destruct after being accessed once. You can create them using either the `/burn/` route or the `X-Burn` header.

**Method 1: Using X-Burn header (recommended):**
```bash
echo "Self-destruct message" | curl -sL --data-binary @- -H "X-Burn: true" http://localhost:8080/
```

**Method 2: Using /burn/ route (backward compatible):**
```bash
echo "Self-destruct message" | curl -sL --data-binary @- http://localhost:8080/burn/
```

**Burn-after-read with other features:**
```bash
# Burn + Base64 encoding
echo "secret" | base64 | curl -sL --data-binary @- \
    -H "X-Burn: true" \
    -H "X-Base64: true" \
    http://localhost:8080/

# Burn + Custom TTL (expires in 1 hour OR after first read, whichever comes first)
echo "Expires soon" | curl -sL --data-binary @- \
    -H "X-Burn: 1" \
    -H "X-TTL: 1h" \
    http://localhost:8080/

# Burn + Custom Slug
echo "Custom burn paste" | curl -sL --data-binary @- \
    -H "X-Burn: yes" \
    -H "X-Slug: MYBURN" \
    http://localhost:8080/
```

**X-Burn accepted values:** `true`, `1`, `yes` (case-sensitive)

Preview/Rendering behavior:

- When viewing a paste via the HTML UI, large content will be previewed instead of fully rendered. The server uses the environment variable `NCLIP_MAX_RENDER_SIZE` to control this behavior. Default value is 262144 (256 KiB). The preview length equals this value.

Example: set preview threshold to 64 KiB

```bash
export NCLIP_MAX_RENDER_SIZE=65536
./nclip
```

Pastes whose size is <= 65536 bytes are rendered inline. Larger pastes show a 64 KiB preview with a link to download the full content.


---

### 7. Base64 Encoding Feature

The `/base64` endpoint (and `X-Base64: true` header) allows you to bypass WAF (Web Application Firewall) or security filters that may block certain content patterns.

**Why use base64 encoding?**
- Bypass WAF blocking shell scripts or curl commands
- Work around API Gateway content filters
- Upload content with special characters that trigger security rules

**How it works:**
1. Encode your content as base64 before uploading
2. Server automatically decodes and stores the original content
3. Content is retrieved in its original (decoded) form

**Supported routes:**
- `POST /base64` - Standard upload with base64
- `POST /` with header `X-Base64: true` - Any route with encoding header

**Encoding variants supported:**
- Standard base64 (RFC 4648)
- URL-safe base64 (RFC 4648)
- Raw (no padding) variants

See `.github/BASE64_FEATURE_SUMMARY.md` for detailed documentation.

---

### 8. Custom Headers Reference

nclip supports several custom HTTP headers to control paste behavior. All headers are optional and can be combined.

**Content Headers:**
- `X-Base64: true` - Upload base64-encoded content. Server decodes before storage. Useful for bypassing WAF/security filters.
- `X-Slug: <custom-id>` - Specify a custom slug/ID for the paste. Must be 3-32 characters, alphanumeric only (A-Z, 2-9, excluding confusing chars: 0, 1, O, I).

**Behavior Headers:**
- `X-TTL: <duration>` - Set custom time-to-live (expiry). Valid range: 1h to 7d. Examples: `2h`, `24h`, `3d`, `7d`
- `X-Burn: <value>` - Enable burn-after-read. Accepted values: `true`, `1`, `yes`. Paste self-destructs after first access.

**Authentication Headers:**
- `Authorization: Bearer <key>` - API key authentication (when `NCLIP_UPLOAD_AUTH=true` on server)
- `X-Api-Key: <key>` - Alternative API key header

**Proxy/HTTPS Detection Headers** (usually set by reverse proxy, not manually):
- `X-Forwarded-Proto: https` - Indicate HTTPS protocol
- `CloudFront-Forwarded-Proto: https` - CloudFront HTTPS indicator
- Other `X-Forwarded-*` headers - Various proxy detection

**Header Combination Examples:**

```bash
# Base64 + Burn + Custom TTL
echo "secret script" | base64 | curl -sL --data-binary @- \
    -H "X-Base64: true" \
    -H "X-Burn: true" \
    -H "X-TTL: 2h" \
    http://localhost:8080/

# Custom Slug + TTL
echo "Important data" | curl -sL --data-binary @- \
    -H "X-Slug: DATA2024" \
    -H "X-TTL: 7d" \
    http://localhost:8080/

# Burn + API Key (when auth enabled)
echo "Confidential" | curl -sL --data-binary @- \
    -H "X-Burn: 1" \
    -H "X-Api-Key: my-secret-key" \
    http://localhost:8080/

# All features combined
cat sensitive.sh | base64 | curl -sL --data-binary @- \
    -H "X-Base64: true" \
    -H "X-Burn: true" \
    -H "X-TTL: 1h" \
    -H "X-Slug: DEPLOY" \
    -H "Authorization: Bearer my-key" \
    http://localhost:8080/
```

See the main README and API documentation for more details.

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

---

### 5. Advanced: Burn-After-Read

**Create burn-after-read paste (curl):**
```bash
echo "Self-destruct message" | curl -sL --data-binary @- http://localhost:8080/burn/
```

---

### 6. Notes on Custom Headers

- `X-TTL`: Set a custom time-to-live (expiry) for a paste. Valid range: 1h–7d. Example: `X-TTL: 2h`
- `X-SLUG`: Specify a custom slug/ID for the paste. Must be unique and valid. Example: `X-SLUG: MYPASTE`
- Other `X-XXX` headers (e.g., `X-Forwarded-Proto`, `X-Scheme`) may be used for proxy detection, HTTPS, and debugging.

See the main README and API documentation for more details.
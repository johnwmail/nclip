package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/config"
)

// WebUIHandler handles web interface
type WebUIHandler struct {
	config *config.Config
}

// NewWebUIHandler creates a new web UI handler
func NewWebUIHandler(config *config.Config) *WebUIHandler {
	return &WebUIHandler{
		config: config,
	}
}

// Index handles the main page via GET /
func (h *WebUIHandler) Index(c *gin.Context) {
	// Use configured URL or derive from request
	baseURL := h.config.URL
	if baseURL == "" {
		// Determine scheme - check for HTTPS indicators
		scheme := "http"
		if h.isHTTPS(c) {
			scheme = "https"
		}
		baseURL = fmt.Sprintf("%s://%s", scheme, c.Request.Host)
	}

	// If request is from CLI tool, return plain text usage examples
	if h.isCli(c) {
		h.serveCLIUsage(c, baseURL)
		return
	}

	// Pass version info and upload-auth flag to template
	c.HTML(http.StatusOK, "index.html", struct {
		Title      string
		Config     struct{ URL string }
		Version    string
		BuildTime  string
		CommitHash string
		UploadAuth bool
	}{
		Title:      "NCLIP - HTTP Clipboard",
		Config:     struct{ URL string }{URL: baseURL},
		Version:    h.config.Version,
		BuildTime:  h.config.BuildTime,
		CommitHash: h.config.CommitHash,
		UploadAuth: h.config.UploadAuth,
	})
}

// isCli detects if the request is from a CLI tool (curl, wget, PowerShell, etc.)
func (h *WebUIHandler) isCli(c *gin.Context) bool {
	userAgent := strings.ToLower(c.Request.Header.Get("User-Agent"))
	acceptHeader := c.Request.Header.Get("Accept")

	// If the client explicitly accepts HTML, treat it as a browser
	if strings.Contains(acceptHeader, "text/html") {
		return false
	}

	// Check for common CLI tools
	cliTools := []string{"curl", "wget", "powershell", "httpie", "invoke-webrequest", "invoke-restmethod"}
	for _, tool := range cliTools {
		if strings.Contains(userAgent, tool) {
			return true
		}
	}

	return false
}

// serveCLIUsage returns plain text usage examples for CLI tools
func (h *WebUIHandler) serveCLIUsage(c *gin.Context, baseURL string) {
	authExamples := ""
	if h.config.UploadAuth {
		authExamples = fmt.Sprintf(`
# With API key authentication (using Authorization header):
echo "Hello World" | curl -sL --data-binary @- -H "Authorization: Bearer YOUR_API_KEY" %s

# With API key authentication (using X-Api-Key header):
echo "Hello World" | curl -sL --data-binary @- -H "X-Api-Key: YOUR_API_KEY" %s
`, baseURL, baseURL)
	}

	usage := fmt.Sprintf(`NCLIP - HTTP Clipboard Service
Version: %s

Usage Examples:
===============

# Upload content:
echo "Hello World" | curl -sL --data-binary @- %s
%s
# Upload a file:
curl -sL --data-binary @/path/to/file.txt %s

# Upload with custom TTL (time-to-live):
echo "Expires in 1 hour" | curl -sL --data-binary @- -H "X-TTL: 1h" %s

# Upload with custom slug:
echo "Custom URL" | curl -sL --data-binary @- -H "X-Slug: my-custom-slug" %s

# Upload with base64 encoding:
echo "Secret data" | base64 | curl -sL --data-binary @- %s/base64

# Create burn-after-read paste (self-destructs after first view):
echo "Secret message" | curl -sL --data-binary @- %s/burn/

# Retrieve content:
curl -sL %s/SLUG              # HTML view
curl -sL %s/raw/SLUG          # Raw content

# Get metadata (JSON):
curl -sL %s/json/SLUG

For more information and web interface, visit: %s
`, h.config.Version, baseURL, authExamples, baseURL, baseURL, baseURL, baseURL, baseURL, baseURL, baseURL, baseURL, baseURL)

	c.String(http.StatusOK, usage)
}

// isHTTPS detects if the original request was HTTPS, even behind proxies
func (h *WebUIHandler) isHTTPS(c *gin.Context) bool {
	// Direct TLS connection
	if c.Request.TLS != nil {
		return true
	}

	// Check common proxy headers for original protocol
	if proto := c.GetHeader("X-Forwarded-Proto"); proto == "https" {
		return true
	}
	if proto := c.GetHeader("X-Forwarded-Protocol"); proto == "https" {
		return true
	}
	if scheme := c.GetHeader("X-Forwarded-Scheme"); scheme == "https" {
		return true
	}
	if scheme := c.GetHeader("X-Scheme"); scheme == "https" {
		return true
	}
	if c.GetHeader("X-Forwarded-Ssl") == "on" {
		return true
	}
	if c.GetHeader("X-Forwarded-Https") == "on" {
		return true
	}

	return false
}

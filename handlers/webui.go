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

// cliTools is the list of common CLI tools to detect in User-Agent headers
var cliTools = []string{"curl", "wget", "powershell", "httpie", "invoke-webrequest", "invoke-restmethod"}

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
	acceptHeader := strings.ToLower(c.Request.Header.Get("Accept"))

	// If the client explicitly accepts HTML, treat it as a browser
	// Check for "text/html" as a complete MIME type to avoid false positives
	if strings.Contains(acceptHeader, "text/html") {
		return false
	}

	// Check for common CLI tools
	for _, tool := range cliTools {
		if strings.Contains(userAgent, tool) {
			return true
		}
	}

	return false
}

// serveCLIUsage returns plain text usage examples for CLI tools
func (h *WebUIHandler) serveCLIUsage(c *gin.Context, baseURL string) {
	var usage string
	if h.config.UploadAuth {
		usage = fmt.Sprintf(`NCLIP - HTTP Clipboard Service
Version: %s

Usage Examples:
===============
Support both Authorization and X-Api-Key headers for API key authentication.

# Upload content (with API key authentication - X-Api-Key header):
echo "Hello World" | curl -sL --data-binary @- -H "X-Api-Key: YOUR_API_KEY" %s

# Upload a file (with API key authentication):
curl -sL --data-binary @/path/to/file.txt -H "Authorization: Bearer YOUR_API_KEY" %s

For more information and web interface, visit: %s
`, h.config.Version, baseURL, baseURL, baseURL) // baseURL repeated for each usage example line
	} else {
		usage = fmt.Sprintf(`NCLIP - HTTP Clipboard Service
Version: %s

Usage Examples:
===============

# Upload content:
echo "Hello World" | curl -sL --data-binary @- %s

# Upload a file:
curl -sL --data-binary @/path/to/file.txt %s

For more information and web interface, visit: %s
`, h.config.Version, baseURL, baseURL, baseURL) // baseURL repeated for each usage example line
	}

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

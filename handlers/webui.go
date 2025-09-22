package handlers

import (
	"fmt"
	"net/http"

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

	// Pass version info to template
	c.HTML(http.StatusOK, "index.html", struct {
		Title      string
		Config     struct{ URL string }
		Version    string
		BuildTime  string
		CommitHash string
	}{
		Title:      "NCLIP - HTTP Clipboard",
		Config:     struct{ URL string }{URL: baseURL},
		Version:    h.config.Version,
		BuildTime:  h.config.BuildTime,
		CommitHash: h.config.CommitHash,
	})
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

package handlers
package handlers

import (
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
	c.HTML(http.StatusOK, "index.html", gin.H{
		"Title": "nclip - HTTP Clipboard",
		"Config": h.config,
	})
}
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SystemHandler handles system endpoints
type SystemHandler struct{}

// NewSystemHandler creates a new system handler
func NewSystemHandler() *SystemHandler {
	return &SystemHandler{}
}

// Health handles health check via GET /health
func (h *SystemHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "nclip",
	})
}

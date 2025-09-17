package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/storage"
	"github.com/johnwmail/nclip/utils"
)

// MetaHandler handles metadata operations
type MetaHandler struct {
	store storage.PasteStore
}

// NewMetaHandler creates a new metadata handler
func NewMetaHandler(store storage.PasteStore) *MetaHandler {
	return &MetaHandler{
		store: store,
	}
}

// GetMetadata handles metadata retrieval via GET /api/v1/meta/:slug
func (h *MetaHandler) GetMetadata(c *gin.Context) {
	slug := c.Param("slug")

	if !utils.IsValidSlug(slug) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid slug format"})
		return
	}

	paste, err := h.store.Get(slug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve paste"})
		return
	}

	if paste == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Paste not found"})
		return
	}

	// Return metadata without content
	c.JSON(http.StatusOK, gin.H{
		"id":              paste.ID,
		"created_at":      paste.CreatedAt,
		"expires_at":      paste.ExpiresAt,
		"size":            paste.Size,
		"content_type":    paste.ContentType,
		"burn_after_read": paste.BurnAfterRead,
		"read_count":      paste.ReadCount,
	})
}

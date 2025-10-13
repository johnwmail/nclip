package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/storage"
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

// GetMetadata handles metadata retrieval via GET /api/v1/meta/:slug and GET /json/:slug
func (h *MetaHandler) GetMetadata(c *gin.Context) {
	slug := c.Param("slug")

	// Allow any slug to be queried; the store will return not-found if it
	// doesn't exist. Slug format validation is performed during paste
	// creation to prevent invalid slugs from being stored.

	paste, err := h.store.Get(slug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve paste"})
		return
	}

	if paste == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Paste not found"})
		return
	}

	// Return metadata without content, pretty-printed JSON
	response := gin.H{
		"id":              paste.ID,
		"created_at":      paste.CreatedAt,
		"expires_at":      paste.ExpiresAt,
		"size":            paste.Size,
		"content_type":    paste.ContentType,
		"burn_after_read": paste.BurnAfterRead,
		"read_count":      paste.ReadCount,
	}

	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal JSON"})
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", jsonBytes)
}

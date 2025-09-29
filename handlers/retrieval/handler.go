package retrieval

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/config"
	"github.com/johnwmail/nclip/internal/services"
	"github.com/johnwmail/nclip/storage"
	"github.com/johnwmail/nclip/utils"
)

// Handler handles paste retrieval operations
type Handler struct {
	service *services.PasteService
	store   storage.PasteStore
	config  *config.Config
}

// NewHandler creates a new retrieval handler
func NewHandler(service *services.PasteService, store storage.PasteStore, config *config.Config) *Handler {
	return &Handler{
		service: service,
		store:   store,
		config:  config,
	}
}

// isHTTPS detects if the request is over HTTPS
func (h *Handler) isHTTPS(c *gin.Context) bool {
	// Check X-Forwarded-Proto header (common with load balancers/proxies)
	if c.GetHeader("X-Forwarded-Proto") == "https" {
		return true
	}

	// AWS Lambda Function URLs may use different headers
	if c.GetHeader("CloudFront-Forwarded-Proto") == "https" {
		return true
	}

	// Check if the original URL scheme can be detected from request URL
	if strings.HasPrefix(c.Request.Header.Get("Referer"), "https://") {
		return true
	}

	return false
}

// isCli detects if the request is from CLI (curl, wget, Invoke-WebRequest, Invoke-RestMethod, etc.)
func (h *Handler) isCli(c *gin.Context) bool {
	userAgent := strings.ToLower(c.Request.Header.Get("User-Agent"))
	if strings.Contains(userAgent, "curl") ||
		strings.Contains(userAgent, "wget") ||
		strings.Contains(userAgent, "powershell") {
		return true
	}
	return false
}

// View handles paste viewing via GET /:slug
func (h *Handler) View(c *gin.Context) {
	slug := c.Param("slug")

	if !utils.IsValidSlug(slug) {
		c.HTML(http.StatusBadRequest, "view.html", gin.H{
			"Title":      "NCLIP - Error",
			"Error":      "Invalid slug format",
			"Version":    h.config.Version,
			"BuildTime":  h.config.BuildTime,
			"CommitHash": h.config.CommitHash,
			"BaseURL":    h.getBaseURL(c),
		})
		return
	}

	paste, err := h.service.GetPaste(slug)
	if err != nil {
		log.Printf("[ERROR] View: %v", err)
		c.HTML(http.StatusNotFound, "view.html", gin.H{
			"Title":      "NCLIP - Not Found",
			"Error":      "Paste not found or deleted",
			"Version":    h.config.Version,
			"BuildTime":  h.config.BuildTime,
			"CommitHash": h.config.CommitHash,
			"BaseURL":    h.getBaseURL(c),
		})
		return
	}

	// Increment read count
	if err := h.service.IncrementReadCount(slug); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to increment read count for %s: %v\n", slug, err)
	}

	content, err := h.service.GetPasteContent(slug)
	if err != nil {
		log.Printf("[ERROR] View: content not found or deleted for slug %s: %v", slug, err)
		c.HTML(http.StatusNotFound, "view.html", gin.H{
			"Title":      "NCLIP - Not Found",
			"Error":      "Paste content not found or deleted",
			"Version":    h.config.Version,
			"BuildTime":  h.config.BuildTime,
			"CommitHash": h.config.CommitHash,
			"BaseURL":    h.getBaseURL(c),
		})
		return
	}

	// If burn-after-read, delete and return 404 if accessed again
	if paste.BurnAfterRead {
		if err := h.service.DeletePaste(slug); err != nil {
			fmt.Printf("Failed to delete burn-after-read paste %s: %v\n", slug, err)
		}
	}
	if h.isCli(c) {
		c.Header("Content-Type", paste.ContentType)
		c.Header("Content-Length", fmt.Sprintf("%d", paste.Size))
		c.Data(http.StatusOK, paste.ContentType, content)
		return
	}
	if strings.Contains(c.Request.Header.Get("Accept"), "text/html") {
		c.HTML(http.StatusOK, "view.html", gin.H{
			"Title":      fmt.Sprintf("NCLIP - Paste %s", paste.ID),
			"Paste":      paste,
			"IsText":     utils.IsTextContent(paste.ContentType),
			"Content":    string(content),
			"Version":    h.config.Version,
			"BuildTime":  h.config.BuildTime,
			"CommitHash": h.config.CommitHash,
			"BaseURL":    h.getBaseURL(c),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":              paste.ID,
		"created_at":      paste.CreatedAt,
		"expires_at":      paste.ExpiresAt,
		"size":            paste.Size,
		"content_type":    paste.ContentType,
		"burn_after_read": paste.BurnAfterRead,
		"content":         string(content),
	})
}

// Raw handles raw content download via GET /raw/:slug
func (h *Handler) Raw(c *gin.Context) {
	slug := c.Param("slug")

	paste, err := h.service.GetPaste(slug)
	if err != nil {
		log.Printf("[ERROR] Raw: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Paste not found or deleted"})
		return
	}

	// Increment read count
	if err := h.service.IncrementReadCount(slug); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to increment read count for %s: %v\n", slug, err)
	}

	content, err := h.service.GetPasteContent(slug)
	if err != nil {
		log.Printf("[ERROR] Raw: content not found or deleted for slug %s: %v", slug, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Paste content not found or deleted"})
		return
	}

	// If burn-after-read, delete and return 404 if accessed again
	if paste.BurnAfterRead {
		if err := h.service.DeletePaste(slug); err != nil {
			fmt.Printf("Failed to delete burn-after-read paste %s: %v\n", slug, err)
		}
		// After deletion, return 404 and no content
		c.JSON(http.StatusNotFound, gin.H{"error": "Paste not found or deleted (burn-after-read)"})
		return
	}
	c.Header("Content-Type", paste.ContentType)
	c.Header("Content-Length", fmt.Sprintf("%d", paste.Size))
	ext := utils.ExtensionByMime(paste.ContentType)
	filename := slug
	if ext != "" {
		filename = slug + ext
	}
	escaped := url.PathEscape(filename)
	if utils.IsTextContent(paste.ContentType) {
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"; filename*=UTF-8''%s", filename, escaped))
	} else {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", filename, escaped))
	}
	c.Data(http.StatusOK, paste.ContentType, content)
}

// getBaseURL returns the base URL for the application
func (h *Handler) getBaseURL(c *gin.Context) string {
	scheme := "http"
	if h.isHTTPS(c) {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, c.Request.Host)
}

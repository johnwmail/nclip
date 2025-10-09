package retrieval

import (
	"errors"
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
		// Prefer HTML for non-CLI (browser) clients; return JSON for CLI/API clients.
		if !h.isCli(c) {
			c.HTML(http.StatusBadRequest, "view.html", gin.H{
				"Title":      "NCLIP - Error",
				"Error":      "Invalid slug format",
				"Version":    h.config.Version,
				"BuildTime":  h.config.BuildTime,
				"CommitHash": h.config.CommitHash,
				"BaseURL":    h.getBaseURL(c),
			})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid slug format"})
		}
		return
	}

	paste, err := h.service.GetPaste(slug)
	if err != nil {
		// Check if paste expired and was deleted
		if errors.Is(err, storage.ErrExpired) {
			log.Printf("[ERROR] View: paste expired and deleted for slug %s", slug)
			h.renderGone(c, "Paste expired and has been deleted")
			return
		}
		// Other errors (paste not found, etc.)
		log.Printf("[ERROR] View: %v", err)
		h.renderNotFound(c, "Paste not found or deleted")
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
		// use same 404 response as missing paste to keep behavior consistent
		h.renderNotFound(c, "Paste not available or deleted")
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
	if !h.isCli(c) {
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
		// Check if paste expired and was deleted
		if errors.Is(err, storage.ErrExpired) {
			log.Printf("[ERROR] Raw: paste expired and deleted for slug %s", slug)
			c.JSON(http.StatusGone, gin.H{"error": "Paste expired and has been deleted"})
			return
		}
		// Other errors (paste not found, etc.)
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

	// If burn-after-read, delete the paste so subsequent accesses return 404.
	// Serve the content for this request (first read) then delete the stored data.
	if paste.BurnAfterRead {
		if err := h.service.DeletePaste(slug); err != nil {
			fmt.Printf("Failed to delete burn-after-read paste %s: %v\n", slug, err)
		}
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

// renderNotFound sends a consistent 404 response. CLI/API clients receive JSON,
// while browser clients receive the HTML view with a friendly message.
func (h *Handler) renderNotFound(c *gin.Context, message string) {
	if h.isCli(c) {
		c.JSON(http.StatusNotFound, gin.H{"error": message})
	} else {
		c.HTML(http.StatusNotFound, "view.html", gin.H{
			"Title":      "NCLIP - Not Found",
			"Error":      message,
			"Version":    h.config.Version,
			"BuildTime":  h.config.BuildTime,
			"CommitHash": h.config.CommitHash,
			"BaseURL":    h.getBaseURL(c),
		})
	}
}

// renderGone sends a consistent 410 response for expired pastes. CLI/API clients receive JSON,
// while browser clients receive the HTML view with a friendly message.
func (h *Handler) renderGone(c *gin.Context, message string) {
	if h.isCli(c) {
		c.JSON(http.StatusGone, gin.H{"error": message})
	} else {
		c.HTML(http.StatusGone, "view.html", gin.H{
			"Title":      "NCLIP - Gone",
			"Error":      message,
			"Version":    h.config.Version,
			"BuildTime":  h.config.BuildTime,
			"CommitHash": h.config.CommitHash,
			"BaseURL":    h.getBaseURL(c),
		})
	}
}

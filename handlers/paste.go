package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/config"
	"github.com/johnwmail/nclip/models"
	"github.com/johnwmail/nclip/storage"
	"github.com/johnwmail/nclip/utils"
)

// PasteHandler handles paste-related operations
type PasteHandler struct {
	store  storage.PasteStore
	config *config.Config
}

// NewPasteHandler creates a new paste handler
func NewPasteHandler(store storage.PasteStore, config *config.Config) *PasteHandler {
	return &PasteHandler{
		store:  store,
		config: config,
	}
}

// generatePasteURL creates the full URL for a paste, detecting HTTPS from proxy headers
func (h *PasteHandler) generatePasteURL(c *gin.Context, slug string) string {
	// If base URL is explicitly set, use it (takes precedence)
	if h.config.URL != "" {
		return fmt.Sprintf("%s/%s", h.config.URL, slug)
	}

	// Determine scheme - check for HTTPS indicators
	scheme := "http"
	if h.isHTTPS(c) {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s/%s", scheme, c.Request.Host, slug)
}

// isHTTPS detects if the original request was HTTPS, even behind proxies
func (h *PasteHandler) isHTTPS(c *gin.Context) bool {
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

// clients detects if the request is from CLI (curl, wget, Invoke-WebRequest, Invoke-RestMethod, etc.)
func (h *PasteHandler) isCli(c *gin.Context) bool {
	userAgent := strings.ToLower(c.Request.Header.Get("User-Agent"))
	if strings.Contains(userAgent, "curl") ||
		strings.Contains(userAgent, "wget") ||
		strings.Contains(userAgent, "powershell") {
		return true
	}
	return false
}

// Upload handles paste upload via POST /
func (h *PasteHandler) Upload(c *gin.Context) {
	var content []byte
	var filename string
	var err error

	// Check if it's a multipart form (file upload)
	if c.Request.Header.Get("Content-Type") != "" &&
		strings.HasPrefix(c.Request.Header.Get("Content-Type"), "multipart/form-data") {

		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
			return
		}
		defer func() { _ = file.Close() }() // Ignore close errors in defer

		filename = header.Filename
		content, err = io.ReadAll(io.LimitReader(file, h.config.BufferSize))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
			return
		}
	} else {
		// Raw content upload
		content, err = io.ReadAll(io.LimitReader(c.Request.Body, h.config.BufferSize))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read content"})
			return
		}
	}

	if len(content) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Empty content"})
		return
	}

	// Generate unique slug
	slug, err := utils.GenerateSlug(h.config.SlugLength)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate slug"})
		return
	}

	// Detect content type
	contentType := utils.DetectContentType(filename, content)

	// Create paste
	expiresAt := time.Now().Add(h.config.DefaultTTL)
	paste := &models.Paste{
		ID:            slug,
		CreatedAt:     time.Now(),
		ExpiresAt:     &expiresAt,
		Size:          int64(len(content)),
		ContentType:   contentType,
		BurnAfterRead: false,
		ReadCount:     0,
		Content:       content,
	}

	// Store paste
	errStore := h.store.Store(paste)
	if errStore != nil {
		// Log the error for Lambda debugging
		fmt.Printf("[ERROR] Failed to store paste: %v\n", errStore)
		// Return detailed error in response (for debugging, remove in production)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to store paste",
			"details": errStore.Error(),
		})
		return
	}

	// Generate URL
	pasteURL := h.generatePasteURL(c, slug)

	// Return URL as plain text for cli tools compatibility
	if h.isCli(c) ||
		c.Request.Header.Get("Accept") == "text/plain" {
		c.String(http.StatusOK, pasteURL+"\n")
		return
	}

	// Return JSON for other clients
	c.JSON(http.StatusOK, gin.H{
		"url":  pasteURL,
		"slug": slug,
	})
}

// UploadBurn handles burn-after-read paste upload via POST /burn/
func (h *PasteHandler) UploadBurn(c *gin.Context) {
	var content []byte
	var filename string
	var err error

	// Check if it's a multipart form (file upload)
	if c.Request.Header.Get("Content-Type") != "" &&
		strings.HasPrefix(c.Request.Header.Get("Content-Type"), "multipart/form-data") {

		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
			return
		}
		defer func() { _ = file.Close() }() // Ignore close errors in defer

		filename = header.Filename
		content, err = io.ReadAll(io.LimitReader(file, h.config.BufferSize))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
			return
		}
	} else {
		// Raw content upload
		content, err = io.ReadAll(io.LimitReader(c.Request.Body, h.config.BufferSize))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read content"})
			return
		}
	}

	if len(content) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Empty content"})
		return
	}

	// Generate unique slug
	slug, err := utils.GenerateSlug(h.config.SlugLength)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate slug"})
		return
	}

	// Detect content type
	contentType := utils.DetectContentType(filename, content)

	// Create burn-after-read paste
	expiresAt := time.Now().Add(h.config.DefaultTTL)
	paste := &models.Paste{
		ID:            slug,
		CreatedAt:     time.Now(),
		ExpiresAt:     &expiresAt,
		Size:          int64(len(content)),
		ContentType:   contentType,
		BurnAfterRead: true,
		ReadCount:     0,
		Content:       content,
	}

	// Store paste
	if err := h.store.Store(paste); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store paste"})
		return
	}

	// Generate URL
	pasteURL := h.generatePasteURL(c, slug)

	// Return URL as plain text for cli tools compatibility
	if h.isCli(c) ||
		c.Request.Header.Get("Accept") == "text/plain" {
		c.String(http.StatusOK, pasteURL+"\n")
		return
	}

	// Return JSON for other clients
	c.JSON(http.StatusOK, gin.H{
		"url":             pasteURL,
		"slug":            slug,
		"burn_after_read": true,
	})
}

// View handles viewing a paste via GET /:slug
func (h *PasteHandler) View(c *gin.Context) {
	slug := c.Param("slug")

	if !utils.IsValidSlug(slug) {
		c.HTML(http.StatusBadRequest, "view.html", gin.H{
			"Title":      "NCLIP - Error",
			"Error":      "Invalid slug format",
			"Version":    h.config.Version,
			"BuildTime":  h.config.BuildTime,
			"CommitHash": h.config.CommitHash,
		})
		return
	}

	paste, err := h.store.Get(slug)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "view.html", gin.H{
			"Title":      "NCLIP - Error",
			"Error":      "Failed to retrieve paste",
			"Version":    h.config.Version,
			"BuildTime":  h.config.BuildTime,
			"CommitHash": h.config.CommitHash,
		})
		return
	}

	if paste == nil {
		c.HTML(http.StatusNotFound, "view.html", gin.H{
			"Title":      "NCLIP - Error",
			"Error":      "Paste not found",
			"Version":    h.config.Version,
			"BuildTime":  h.config.BuildTime,
			"CommitHash": h.config.CommitHash,
		})
		return
	}

	// Increment read count
	if err := h.store.IncrementReadCount(slug); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to increment read count for %s: %v\n", slug, err)
	}

	// If this is a burn-after-read paste, delete it after reading
	if paste.BurnAfterRead {
		if err := h.store.Delete(slug); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Failed to delete burn-after-read paste %s: %v\n", slug, err)
		}
	}

	// Check if this is a cli tools request - serve raw content directly
	if h.isCli(c) {
		// Serve raw content directly instead of redirecting
		c.Header("Content-Type", paste.ContentType)
		c.Header("Content-Length", fmt.Sprintf("%d", paste.Size))
		c.Data(http.StatusOK, paste.ContentType, paste.Content)
		return
	}

	// Return HTML view for browsers
	if strings.Contains(c.Request.Header.Get("Accept"), "text/html") {
		c.HTML(http.StatusOK, "view.html", gin.H{
			"Title":      fmt.Sprintf("NCLIP - Paste %s", paste.ID),
			"Paste":      paste,
			"IsText":     utils.IsTextContent(paste.ContentType),
			"Content":    string(paste.Content),
			"Version":    h.config.Version,
			"BuildTime":  h.config.BuildTime,
			"CommitHash": h.config.CommitHash,
		})
		return
	}

	// Return JSON for API clients
	c.JSON(http.StatusOK, gin.H{
		"id":              paste.ID,
		"created_at":      paste.CreatedAt,
		"expires_at":      paste.ExpiresAt,
		"size":            paste.Size,
		"content_type":    paste.ContentType,
		"burn_after_read": paste.BurnAfterRead,
		"content":         string(paste.Content),
	})
}

// Raw handles raw content download via GET /raw/:slug
func (h *PasteHandler) Raw(c *gin.Context) {
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

	// Increment read count
	if err := h.store.IncrementReadCount(slug); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to increment read count for %s: %v\n", slug, err)
	}

	// If this is a burn-after-read paste, delete it after reading
	if paste.BurnAfterRead {
		if err := h.store.Delete(slug); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Failed to delete burn-after-read paste %s: %v\n", slug, err)
		}
	}

	// Set appropriate headers
	c.Header("Content-Type", paste.ContentType)
	c.Header("Content-Length", fmt.Sprintf("%d", paste.Size))

	// Suggest filename with extension based on MIME type
	ext := utils.ExtensionByMime(paste.ContentType)
	filename := slug
	if ext != "" {
		filename = slug + ext
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// Return raw content
	c.Data(http.StatusOK, paste.ContentType, paste.Content)
}

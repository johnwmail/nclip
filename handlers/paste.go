package handlers

import (
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
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
	// Add panic recovery to always return debug info
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			h.writeError(c, http.StatusInternalServerError, "Panic in upload handler", fmt.Sprintf("panic: %v\nstack: %s", r, stack))
		}
	}()

	fmt.Printf("[DEBUG] Handler: method=%s, content-type=%s, content-length=%s, UA=%s\n", c.Request.Method, c.Request.Header.Get("Content-Type"), c.Request.Header.Get("Content-Length"), c.Request.Header.Get("User-Agent"))
	fmt.Printf("[DEBUG] All headers: %v\n", c.Request.Header)
	fmt.Printf("[DEBUG] Upload handler invoked at %v\n", time.Now())
	fmt.Printf("[DEBUG] RemoteAddr: %s, Method: %s\n", c.Request.RemoteAddr, c.Request.Method)

	var content []byte
	var filename string
	var err error
	debugInfo := func(extra string) string {
		return fmt.Sprintf("[DEBUG] Method: %s\n[DEBUG] RemoteAddr: %s\n[DEBUG] Content-Type: %s\n[DEBUG] Content-Length: %s\n[DEBUG] Headers: %v\n%s",
			c.Request.Method,
			c.Request.RemoteAddr,
			c.Request.Header.Get("Content-Type"),
			c.Request.Header.Get("Content-Length"),
			c.Request.Header,
			extra,
		)
	}

	if c.Request.Header.Get("Content-Type") != "" && strings.HasPrefix(c.Request.Header.Get("Content-Type"), "multipart/form-data") {
		fmt.Printf("[DEBUG] Upload path: multipart/form-data\n")
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			fmt.Printf("[ERROR] FormFile error: %v\n", err)
			h.writeError(c, http.StatusBadRequest, "No file provided (multipart)", debugInfo(err.Error()))
			return
		}
		defer func() { _ = file.Close() }()
		filename = header.Filename
		content, err = io.ReadAll(io.LimitReader(file, h.config.BufferSize))
		fmt.Printf("[DEBUG] Multipart: filename=%s, bytes_read=%d\n", filename, len(content))
		if err != nil {
			fmt.Printf("[ERROR] io.ReadAll error: %v\n", err)
			h.writeError(c, http.StatusInternalServerError, "Failed to read file (multipart)", debugInfo(err.Error()))
			return
		}
		if len(content) > 0 {
			fmt.Printf("[DEBUG] Multipart: first 64 bytes: % x\n", content[:min(64, len(content))])
		}
	} else {
		fmt.Printf("[DEBUG] Upload path: raw body\n")
		if c.Request.Body == nil {
			fmt.Printf("[ERROR] Request.Body is nil\n")
		}
		content, err = io.ReadAll(io.LimitReader(c.Request.Body, h.config.BufferSize))
		fmt.Printf("[DEBUG] Raw: bytes_read=%d\n", len(content))
		if err != nil {
			fmt.Printf("[ERROR] io.ReadAll (raw) error: %v\n", err)
			h.writeError(c, http.StatusInternalServerError, "Failed to read content (raw)", debugInfo(err.Error()))
			return
		}
		if len(content) > 0 {
			fmt.Printf("[DEBUG] Raw: first 64 bytes: % x\n", content[:min(64, len(content))])
		}
		filename = c.Request.Header.Get("X-Filename")
		if filename == "" {
			filename = "upload"
		}
	}

	clen := c.Request.Header.Get("Content-Length")
	if len(content) == 0 && clen != "" && clen != "0" {
		fmt.Printf("[WARN] Content is empty but Content-Length header is %s\n", clen)
		h.writeError(c, http.StatusBadRequest, "Empty content (body empty but Content-Length > 0)", debugInfo("No data provided in upload; Content-Length="+clen))
		return
	}

	if len(content) == 0 {
		fmt.Printf("[ERROR] Empty content in upload\n")
		h.writeError(c, http.StatusBadRequest, "Empty content", debugInfo("No data provided in upload"))
		return
	}

	// Log content size for debugging (chunking handled in storage layer)
	fmt.Printf("[DEBUG] Content size: %d bytes\n", len(content))

	fmt.Printf("[DEBUG] Detected filename: %s, content-type: %s\n", filename, http.DetectContentType(content))

	// Generate unique slug
	slug, err := utils.GenerateSlug(h.config.SlugLength)
	if err != nil {
		fmt.Printf("[ERROR] Failed to generate slug: %v\n", err)
		h.writeError(c, http.StatusInternalServerError, "Failed to generate slug", debugInfo(err.Error()))
		return
	}

	// Detect content type
	contentType := utils.DetectContentType(filename, content)
	fmt.Printf("[DEBUG] utils.DetectContentType: %s\n", contentType)

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

       // Store paste with extra debug output
       fmt.Printf("[DEBUG] About to store paste: slug=%s, content_size=%d, backend=%T, config.BufferSize=%d\n", slug, len(content), h.store, h.config.BufferSize)
       errStore := h.store.Store(paste)
       if errStore != nil {
	       // Log the error for Lambda debugging
	       fmt.Printf("[ERROR] Failed to store paste: %v\n", errStore)
	       fmt.Printf("[ERROR] Paste details: slug=%s, content_size=%d, backend=%T, config=%+v\n", slug, len(content), h.store, h.config)
	       // Print stack trace for debugging
	       fmt.Printf("[DEBUG] Stack trace: %s\n", debugStack())
	       // Return error details to client for debugging (temporarily)
	       h.writeError(c, http.StatusInternalServerError, "Failed to store paste", debugInfo(errStore.Error()+" | stack: "+debugStack()))
	       return
       }
       fmt.Printf("[DEBUG] Successfully stored paste: slug=%s, content_size=%d\n", slug, len(content))

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

// writeError returns a plain text error for CLI clients, JSON for others
func (h *PasteHandler) writeError(c *gin.Context, status int, errorMsg, details string) {
	userAgent := strings.ToLower(c.Request.Header.Get("User-Agent"))
	isCli := strings.Contains(userAgent, "curl") || strings.Contains(userAgent, "wget") || strings.Contains(userAgent, "powershell")
	if isCli || c.Request.Header.Get("Accept") == "text/plain" {
		c.String(status, "%s: %s\n", errorMsg, details)
	} else {
		c.JSON(status, gin.H{"error": errorMsg, "details": details})
	}
}

// min returns the smaller of a and b
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// debugStack returns a string stack trace for debugging
func debugStack() string {
	return string(debug.Stack())
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided", "details": err.Error()})
			return
		}
		defer func() { _ = file.Close() }() // Ignore close errors in defer
		filename = header.Filename
		content, err = io.ReadAll(io.LimitReader(file, h.config.BufferSize))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file", "details": err.Error()})
			return
		}
	} else {
		// Raw content upload
		content, err = io.ReadAll(io.LimitReader(c.Request.Body, h.config.BufferSize))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read content", "details": err.Error()})
			return
		}
	}

	if len(content) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Empty content", "details": "No data provided in upload"})
		return
	}

	// Generate unique slug
	slug, err := utils.GenerateSlug(h.config.SlugLength)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate slug", "details": err.Error()})
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
		fmt.Printf("[ERROR] Failed to store paste: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store paste", "details": err.Error()})
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

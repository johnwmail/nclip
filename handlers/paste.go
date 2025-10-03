package handlers

import (
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/config"
	"github.com/johnwmail/nclip/models"
	"github.com/johnwmail/nclip/storage"
	"github.com/johnwmail/nclip/utils"
)

// Helper: readUploadContent extracts content, filename, and content-type from request
func (h *PasteHandler) readUploadContent(c *gin.Context) ([]byte, string, string, error) {
	limit := h.config.BufferSize
	contentTypeHeader := c.Request.Header.Get("Content-Type")
	if contentTypeHeader != "" && strings.HasPrefix(contentTypeHeader, "multipart/form-data") {
		return h.readMultipartUpload(c, limit)
	}
	return h.readDirectUpload(c, limit)
}

func (h *PasteHandler) readMultipartUpload(c *gin.Context, limit int64) ([]byte, string, string, error) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		return nil, "", "", fmt.Errorf("no file provided")
	}
	defer func() { _ = file.Close() }()

	filename := header.Filename
	if header.Size > 0 && header.Size > limit {
		return nil, filename, "", fmt.Errorf("content too large: %d bytes exceeds limit of %d bytes", header.Size, limit)
	}

	content, exceeded, err := h.readLimitedContent(file)
	if err != nil {
		return nil, filename, "", fmt.Errorf("failed to read file")
	}
	if exceeded {
		return nil, filename, "", fmt.Errorf("content too large: exceeds limit of %d bytes", limit)
	}

	contentType := utils.DetectContentType(filename, content)
	if len(content) == 0 {
		return nil, filename, contentType, fmt.Errorf("empty content")
	}
	return content, filename, contentType, nil
}

func (h *PasteHandler) readDirectUpload(c *gin.Context, limit int64) ([]byte, string, string, error) {
	if contentLength := c.Request.ContentLength; contentLength > 0 && contentLength > limit {
		return nil, "", "", fmt.Errorf("content too large: %d bytes exceeds limit of %d bytes", contentLength, limit)
	}

	content, exceeded, err := h.readLimitedContent(c.Request.Body)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to read content")
	}
	if exceeded {
		return nil, "", "", fmt.Errorf("content too large: exceeds limit of %d bytes", limit)
	}

	contentType := ""
	ctHeader := c.ContentType()
	if utils.IsDebugEnabled() {
		log.Printf("[DEBUG] ContentType header: %s", ctHeader)
	}
	if ctHeader != "" {
		if parsedType, _, err := mime.ParseMediaType(ctHeader); err == nil {
			contentType = parsedType
			if utils.IsDebugEnabled() {
				log.Printf("[DEBUG] Parsed contentType: %s", contentType)
			}
		} else {
			contentType = ctHeader
		}
	}
	if contentType == "" {
		contentType = utils.DetectContentType("", content)
	}

	if len(content) == 0 {
		return nil, "", contentType, fmt.Errorf("empty content")
	}
	return content, "", contentType, nil
}

func (h *PasteHandler) readLimitedContent(r io.Reader) ([]byte, bool, error) {
	if r == nil {
		return nil, false, fmt.Errorf("nil reader")
	}

	limit := h.config.BufferSize
	if limit <= 0 {
		return nil, false, fmt.Errorf("invalid buffer size configuration: %d", limit)
	}

	buf, err := io.ReadAll(io.LimitReader(r, limit))
	if err != nil {
		return nil, false, err
	}
	if int64(len(buf)) < limit {
		return buf, false, nil
	}

	var extra [1]byte
	n, err := r.Read(extra[:])
	if n > 0 {
		return nil, true, nil
	}
	if err != nil && err != io.EOF {
		return nil, false, err
	}
	return buf, false, nil
}
func (h *PasteHandler) generateUniqueSlug() (string, error) {
	batchSize := 5
	lengths := []int{5, 6, 7}
	var slug string
	for _, length := range lengths {
		candidates, err := utils.GenerateSlugBatch(batchSize, length)
		if err != nil {
			return "", fmt.Errorf("failed to generate slug")
		}
		for _, candidate := range candidates {
			exists, err := h.store.Exists(candidate)
			if err != nil {
				continue // skip on error
			}
			if !exists {
				slug = candidate
				return slug, nil
			}
			// exists, check if expired
			existing, err := h.store.Get(candidate)
			if err != nil || existing == nil || existing.IsExpired() {
				slug = candidate
				return slug, nil
			}
		}
	}
	return "", fmt.Errorf("failed to generate unique slug after 3 batches")
}

// Helper: parseBurnTTL parses TTL for burn-after-read
func (h *PasteHandler) parseBurnTTL(c *gin.Context) (time.Time, error) {
	ttlStr := c.GetHeader("X-TTL")
	if ttlStr != "" {
		d, err := time.ParseDuration(ttlStr)
		minTTL := time.Hour
		maxTTL := 7 * 24 * time.Hour
		if utils.IsDebugEnabled() {
			log.Printf("[DEBUG] Parsed X-TTL duration: %v (raw: %s)", d, ttlStr)
		}
		if err != nil || d < minTTL || d > maxTTL {
			return time.Time{}, fmt.Errorf("X-TTL must be between 1h and 7d")
		}
		return time.Now().Add(d), nil
	}
	return time.Now().Add(h.config.DefaultTTL), nil
}

// Helper: respondError sends a JSON error response
func respondError(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": msg})
}

// PasteHandler handles paste-related operations
type PasteHandler struct {
	store             storage.PasteStore
	config            *config.Config
	GenerateSlugBatch func(batchSize, length int) ([]string, error)
}

// Helper: readUploadContent extracts content, filename, and content-type from request
// DUPLICATE FUNCTION - REMOVED
/*
func (h *PasteHandler) readUploadContent(c *gin.Context) ([]byte, string, string, error) {
	var content []byte
	var filename string
	var contentType string
	var err error

	// For direct POST requests, check Content-Length to reject obviously large uploads early
	// Skip this check for multipart uploads since Content-Length includes multipart overhead
	isMultipart := c.Request.Header.Get("Content-Type") != "" && strings.HasPrefix(c.Request.Header.Get("Content-Type"), "multipart/form-data")
	log.Printf("[DEBUG] ContentLength: %d, isMultipart: %v", c.Request.ContentLength, isMultipart)
	if !isMultipart {
		if contentLength := c.Request.ContentLength; contentLength > 0 && contentLength > h.config.BufferSize {
			return nil, "", "", fmt.Errorf("content too large: %d bytes exceeds limit of %d bytes", contentLength, h.config.BufferSize)
		}
	}

	if isMultipart {
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			return nil, "", "", fmt.Errorf("no file provided")
		}
		defer func() { _ = file.Close() }()
		filename = header.Filename
		content, err = io.ReadAll(&countingReader{r: file, limit: h.config.BufferSize})
		if err != nil {
			return nil, filename, "", err
		}
		// For multipart uploads, detect content type from file content
		contentType = utils.DetectContentType(filename, content)
	} else {
		content, err = io.ReadAll(&countingReader{r: c.Request.Body, limit: h.config.BufferSize})
		if err != nil {
			return nil, "", "", err
		}
		// For direct POST, use the Content-Type header if provided
		contentTypeHeader := c.ContentType()
		log.Printf("[DEBUG] ContentType header: %s", contentTypeHeader)
		if contentTypeHeader != "" {
			// Parse the media type to extract just the MIME type (without parameters)
			if parsedType, _, err := mime.ParseMediaType(contentTypeHeader); err == nil {
				contentType = parsedType
				log.Printf("[DEBUG] Parsed contentType: %s", contentType)
			} else {
				contentType = contentTypeHeader
			}
		}
		// TEMP: Force content type for testing
		if contentType == "" {
			contentType = "text/javascript"
			log.Printf("[DEBUG] Forced contentType to: %s", contentType)
		}
	}
	if len(content) == 0 {
		return nil, filename, contentType, fmt.Errorf("empty content")
	}
	return content, filename, contentType, nil
}
*/

// respondUploadError handles upload error responses with appropriate status codes
func (h *PasteHandler) respondUploadError(c *gin.Context, err error) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	// Use 413 Payload Too Large for size limit violations
	if strings.Contains(err.Error(), "content too large") {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": err.Error()})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

// storeContentAndMetadata stores both content and metadata for a paste
func (h *PasteHandler) storeContentAndMetadata(slug string, content []byte, paste *models.Paste) error {
	if err := h.store.StoreContent(slug, content); err != nil {
		return fmt.Errorf("failed to store content: %w", err)
	}
	if err := h.store.Store(paste); err != nil {
		return fmt.Errorf("failed to store metadata: %w", err)
	}
	return nil
}

// respondWithPasteURL generates and returns the appropriate response for a paste URL
func (h *PasteHandler) respondWithPasteURL(c *gin.Context, slug string, burnAfterRead bool) {
	pasteURL := h.generatePasteURL(c, slug)

	// Return URL as plain text for cli tools compatibility
	if h.isCli(c) || c.Request.Header.Get("Accept") == "text/plain" {
		c.String(http.StatusOK, pasteURL+"\n")
		return
	}

	// Return JSON for other clients
	c.JSON(http.StatusOK, gin.H{
		"url":             pasteURL,
		"slug":            slug,
		"burn_after_read": burnAfterRead,
	})
}
func (h *PasteHandler) selectOrGenerateSlug(c *gin.Context) (string, error) {
	batchSize := 5
	lengths := []int{5, 6, 7}
	var slug string
	var lastCandidates []string
	var lastCollisions []string
	found := false
	for _, length := range lengths {
		candidates, err := h.GenerateSlugBatch(batchSize, length)
		if err != nil {
			return "", fmt.Errorf("failed to generate slug")
		}
		lastCandidates = candidates
		lastCollisions = nil
		for _, candidate := range candidates {
			existing, err := h.store.Get(candidate)
			if err != nil || existing == nil || existing.IsExpired() {
				slug = candidate
				found = true
				break
			} else {
				lastCollisions = append(lastCollisions, candidate)
			}
		}
		if found {
			break
		}
	}
	if !found {
		log.Printf("[ERROR] Could not generate unique slug after 3 batches. Last candidates: %v. Collisions: %v", lastCandidates, lastCollisions)
		return "", fmt.Errorf("failed to generate unique slug after 3 batches")
	}
	return slug, nil
}

// Helper: parseTTL parses TTL from X-TTL header or uses default
func (h *PasteHandler) parseTTL(c *gin.Context) (time.Time, error) {
	ttlStr := c.GetHeader("X-TTL")
	if ttlStr != "" {
		d, err := time.ParseDuration(ttlStr)
		minTTL := time.Hour
		maxTTL := 7 * 24 * time.Hour
		if utils.IsDebugEnabled() {
			log.Printf("[DEBUG] Parsed X-TTL duration: %v (raw: %s)", d, ttlStr)
		}
		if err != nil || d < minTTL || d > maxTTL {
			return time.Time{}, fmt.Errorf("X-TTL must be between 1h and 7d")
		}
		return time.Now().Add(d), nil
	}
	return time.Now().Add(h.config.DefaultTTL), nil
}

// Helper: storePasteAndRespond stores paste and responds to client
func (h *PasteHandler) storePasteAndRespond(c *gin.Context, slug string, content []byte, expiresAt time.Time, filename string, contentType string) {
	if contentType == "" {
		contentType = utils.DetectContentType(filename, content)
	}
	// Only set BurnAfterRead true for /burn/ endpoint
	burnAfterRead := strings.HasSuffix(c.FullPath(), "/burn/")
	paste := &models.Paste{
		ID:            slug,
		CreatedAt:     time.Now(),
		ExpiresAt:     &expiresAt,
		Size:          int64(len(content)),
		ContentType:   contentType,
		BurnAfterRead: burnAfterRead,
		ReadCount:     0,
	}
	if err := h.store.StoreContent(slug, content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store content"})
		return
	}
	if err := h.store.Store(paste); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store metadata"})
		return
	}
	pasteURL := h.generatePasteURL(c, slug)
	// Always return JSON for web UI (browser)
	if h.isCli(c) || c.Request.Header.Get("Accept") == "text/plain" {
		c.String(http.StatusOK, pasteURL+"\n")
		return
	}
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.JSON(http.StatusOK, gin.H{
		"url":             pasteURL,
		"slug":            slug,
		"burn_after_read": true,
	})
}

// NewPasteHandler creates a new paste handler
func NewPasteHandler(store storage.PasteStore, config *config.Config) *PasteHandler {
	return &PasteHandler{
		store:             store,
		config:            config,
		GenerateSlugBatch: utils.GenerateSlugBatch,
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
	content, filename, contentType, err := h.readUploadContent(c)
	if err != nil {
		log.Printf("[ERROR] %v", err)
		c.Header("Content-Type", "application/json; charset=utf-8")
		// Use 413 Payload Too Large for size limit violations
		if strings.Contains(err.Error(), "content too large") {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	// Check for custom slug header
	customSlug := c.GetHeader("X-Slug")
	if customSlug != "" {
		// Validate slug format
		if !utils.IsValidSlug(customSlug) {
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid slug format"})
			return
		}
		// Check for collision
		exists, err := h.store.Exists(customSlug)
		if err != nil {
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check slug"})
			return
		}
		if exists {
			existing, err := h.store.Get(customSlug)
			if err != nil {
				c.Header("Content-Type", "application/json; charset=utf-8")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve paste"})
				return
			}
			if existing != nil && !existing.IsExpired() {
				c.Header("Content-Type", "application/json; charset=utf-8")
				c.JSON(http.StatusBadRequest, gin.H{"error": "Slug already exists"})
				return
			}
		}
		slug := customSlug
		expiresAt, err := h.parseTTL(c)
		if err != nil {
			log.Printf("[ERROR] %v", err)
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.storePasteAndRespond(c, slug, content, expiresAt, filename, contentType)
		return
	}
	// No custom slug, generate one
	slug, err := h.selectOrGenerateSlug(c)
	if err != nil {
		log.Printf("[ERROR] %v", err)
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	expiresAt, err := h.parseTTL(c)
	if err != nil {
		log.Printf("[ERROR] %v", err)
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.storePasteAndRespond(c, slug, content, expiresAt, filename, contentType)
}

// UploadBurn handles burn-after-read paste upload via POST /burn/
func (h *PasteHandler) UploadBurn(c *gin.Context) {
	content, filename, _, err := h.readUploadContent(c)
	if err != nil {
		log.Printf("[ERROR] %v", err)
		h.respondUploadError(c, err)
		return
	}

	// Detect content type (burn-after-read doesn't use custom content type from header)
	contentType := utils.DetectContentType(filename, content)

	// Parse TTL for burn-after-read
	expiresAt, err := h.parseBurnTTL(c)
	if err != nil {
		log.Printf("[ERROR] %v", err)
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Generate unique slug
	slug, err := h.generateUniqueSlug()
	if err != nil {
		log.Printf("[ERROR] %v", err)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Create and store paste
	paste := &models.Paste{
		ID:            slug,
		CreatedAt:     time.Now(),
		ExpiresAt:     &expiresAt,
		Size:          int64(len(content)),
		ContentType:   contentType,
		BurnAfterRead: true,
		ReadCount:     0,
	}

	if err := h.storeContentAndMetadata(slug, content, paste); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Generate response
	h.respondWithPasteURL(c, slug, true)
}

// View handles viewing a paste via GET /:slug
func (h *PasteHandler) View(c *gin.Context) {
	slug := c.Param("slug")

	// Compute base URL for curl examples
	baseURL := h.config.URL
	if baseURL == "" {
		scheme := "http"
		if h.isHTTPS(c) {
			scheme = "https"
		}
		baseURL = fmt.Sprintf("%s://%s", scheme, c.Request.Host)
	}
	if !utils.IsValidSlug(slug) {
		c.HTML(http.StatusBadRequest, "view.html", gin.H{
			"Title":      "NCLIP - Error",
			"Error":      "Invalid slug format",
			"Version":    h.config.Version,
			"BuildTime":  h.config.BuildTime,
			"CommitHash": h.config.CommitHash,
			"BaseURL":    baseURL,
		})
		return
	}

	paste, err := h.store.Get(slug)
	if err != nil || paste == nil {
		log.Printf("[ERROR] View: paste not found, deleted, or expired for slug %s: %v", slug, err)
		c.HTML(http.StatusNotFound, "view.html", gin.H{
			"Title":      "NCLIP - Not Found",
			"Error":      "Paste not found or deleted",
			"Version":    h.config.Version,
			"BuildTime":  h.config.BuildTime,
			"CommitHash": h.config.CommitHash,
			"BaseURL":    baseURL,
		})
		return
	}

	// Check expiration
	if paste.IsExpired() {
		log.Printf("[ERROR] View: paste expired for slug %s", slug)
		c.HTML(http.StatusNotFound, "view.html", gin.H{
			"Title":      "NCLIP - Not Found",
			"Error":      "Paste not found or deleted (expired)",
			"Version":    h.config.Version,
			"BuildTime":  h.config.BuildTime,
			"CommitHash": h.config.CommitHash,
			"BaseURL":    baseURL,
		})
		return
	}

	// Increment read count
	if err := h.store.IncrementReadCount(slug); err != nil {
		fmt.Printf("Failed to increment read count for %s: %v\n", slug, err)
	}
	var content []byte
	content, err = h.store.GetContent(slug)
	if err != nil {
		log.Printf("[ERROR] View: content not found or deleted for slug %s: %v", slug, err)
		c.HTML(http.StatusNotFound, "view.html", gin.H{
			"Title":      "NCLIP - Not Found",
			"Error":      "Paste content not found or deleted",
			"Version":    h.config.Version,
			"BuildTime":  h.config.BuildTime,
			"CommitHash": h.config.CommitHash,
		})
		return
	}
	if paste.BurnAfterRead {
		if err := h.store.Delete(slug); err != nil {
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
			"BaseURL":    baseURL,
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
func (h *PasteHandler) Raw(c *gin.Context) {
	slug := c.Param("slug")

	if !utils.IsValidSlug(slug) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid slug format"})
		return
	}

	paste, err := h.store.Get(slug)
	if err != nil || paste == nil {
		log.Printf("[ERROR] Raw: paste not found, deleted, or expired for slug %s: %v", slug, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Paste not found or deleted"})
		return
	}

	// Check expiration
	if paste.IsExpired() {
		log.Printf("[ERROR] Raw: paste expired for slug %s", slug)
		c.JSON(http.StatusNotFound, gin.H{"error": "Paste not found or deleted (expired)"})
		return
	}

	// Increment read count
	if err := h.store.IncrementReadCount(slug); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to increment read count for %s: %v\n", slug, err)
	}

	var content []byte
	content, err = h.store.GetContent(slug)
	if err != nil {
		log.Printf("[ERROR] Raw: content not found or deleted for slug %s: %v", slug, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Paste content not found or deleted"})
		return
	}
	// If burn-after-read, delete and return 404 if accessed again
	if paste.BurnAfterRead {
		if err := h.store.Delete(slug); err != nil {
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

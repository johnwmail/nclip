package upload

import (
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/config"
	"github.com/johnwmail/nclip/internal/services"
	"github.com/johnwmail/nclip/utils"
)

// Handler handles paste upload operations
type Handler struct {
	service *services.PasteService
	config  *config.Config
}

// NewHandler creates a new upload handler
func NewHandler(service *services.PasteService, config *config.Config) *Handler {
	return &Handler{
		service: service,
		config:  config,
	}
}

// parseTTL parses TTL from X-TTL header or uses default
func (h *Handler) parseTTL(c *gin.Context) (time.Time, error) {
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

// readUploadContent extracts content, filename, and content-type from request
func (h *Handler) readUploadContent(c *gin.Context) ([]byte, string, string, error) {
	limit := h.config.BufferSize
	contentTypeHeader := c.Request.Header.Get("Content-Type")
	isMultipart := contentTypeHeader != "" && strings.HasPrefix(contentTypeHeader, "multipart/form-data")

	if isMultipart {
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
	if ct := c.ContentType(); ct != "" {
		if parsedType, _, err := mime.ParseMediaType(ct); err == nil {
			contentType = parsedType
		} else {
			contentType = ct
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

func (h *Handler) readLimitedContent(r io.Reader) ([]byte, bool, error) {
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

// storePasteAndRespond stores paste and responds to client
func (h *Handler) storePasteAndRespond(c *gin.Context, req services.CreatePasteRequest) {
	resp, err := h.service.CreatePaste(req)
	if err != nil {
		// Check if this is a validation error (should return 400) or server error (500)
		errMsg := err.Error()
		if strings.Contains(errMsg, "slug already exists") ||
			strings.Contains(errMsg, "invalid slug format") ||
			strings.Contains(errMsg, "X-TTL must be between") {
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
			return
		}
		// For other errors, return 500
		log.Printf("[ERROR] Failed to create paste: %v", err)
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create paste"})
		return
	}

	pasteURL := h.generatePasteURL(c, resp.Slug)
	resp.URL = pasteURL

	// Always return JSON for web UI (browser)
	if h.isCli(c) || c.Request.Header.Get("Accept") == "text/plain" {
		c.String(http.StatusOK, pasteURL+"\n")
		return
	}
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.JSON(http.StatusOK, gin.H{
		"url":             resp.URL,
		"slug":            resp.Slug,
		"burn_after_read": req.BurnAfterRead,
	})
}

// generatePasteURL generates the full URL for a paste
func (h *Handler) generatePasteURL(c *gin.Context, slug string) string {
	scheme := "http"
	if h.isHTTPS(c) {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s", scheme, c.Request.Host, slug)
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

// Upload handles paste upload via POST /
func (h *Handler) Upload(c *gin.Context) {
	content, filename, contentType, err := h.readUploadContent(c)
	if err != nil {
		log.Printf("[ERROR] %v", err)
		c.Header("Content-Type", "application/json; charset=utf-8")
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "content too large") {
			status = http.StatusRequestEntityTooLarge
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	req := services.CreatePasteRequest{
		Content:       content,
		Filename:      filename,
		ContentType:   contentType,
		BurnAfterRead: strings.HasSuffix(c.FullPath(), "/burn/"),
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
		req.CustomSlug = customSlug
	}

	// Parse TTL
	ttl, err := h.parseTTL(c)
	if err != nil {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.TTL = time.Until(ttl)

	h.storePasteAndRespond(c, req)
}

// UploadBurn handles paste upload with burn-after-read via POST /burn/
func (h *Handler) UploadBurn(c *gin.Context) {
	content, filename, contentType, err := h.readUploadContent(c)
	if err != nil {
		log.Printf("[ERROR] %v", err)
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req := services.CreatePasteRequest{
		Content:       content,
		Filename:      filename,
		ContentType:   contentType,
		BurnAfterRead: true,
	}

	expiresAt, err := h.parseTTL(c)
	if err != nil {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.TTL = time.Until(expiresAt)

	h.storePasteAndRespond(c, req)
}

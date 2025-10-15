package upload

import (
	"encoding/base64"
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

// headerEnabled returns true if the given header key is present and not
// explicitly disabled. Presence with an empty value counts as enabled.
// Explicit disabling values (case-insensitive): "0", "false", "no".
func headerEnabled(c *gin.Context, header string) bool {
	vals, ok := c.Request.Header[header]
	if !ok {
		return false
	}
	if len(vals) == 0 {
		return true
	}
	// Scan all values for explicit disabling tokens
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v == "" {
			continue // empty value counts as enabled, keep scanning
		}
		lv := strings.ToLower(v)
		if lv == "0" || lv == "false" || lv == "no" {
			return false
		}
	}
	return true
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
// Supports X-Base64 header for base64 encoded content
func (h *Handler) readUploadContent(c *gin.Context) ([]byte, string, string, error) {
	limit := h.config.BufferSize
	contentTypeHeader := c.Request.Header.Get("Content-Type")

	var content []byte
	var filename string
	var contentType string
	var err error

	if contentTypeHeader != "" && strings.HasPrefix(contentTypeHeader, "multipart/form-data") {
		content, filename, contentType, err = h.readMultipartUpload(c, limit)
	} else {
		content, filename, contentType, err = h.readDirectUpload(c, limit)
	}

	if err != nil {
		return nil, filename, contentType, err
	}

	// Check if content is base64 encoded
	// Semantics: header presence enables base64 unless explicitly set to 0/false/no
	if headerEnabled(c, "X-Base64") {
		decoded, decodeErr := h.decodeBase64Content(content)
		if decodeErr != nil {
			return nil, filename, contentType, decodeErr
		}

		// Validate decoded content size
		if int64(len(decoded)) > limit {
			return nil, filename, contentType, fmt.Errorf("decoded content too large: %d bytes exceeds limit of %d bytes", len(decoded), limit)
		}

		// Validate decoded content is not empty
		if len(decoded) == 0 {
			return nil, filename, contentType, fmt.Errorf("decoded content is empty")
		}

		// Re-detect content type based on decoded content
		contentType = utils.DetectContentType(filename, decoded)

		if utils.IsDebugEnabled() {
			log.Printf("[DEBUG] Base64 decoded: %d bytes → %d bytes", len(content), len(decoded))
		}

		return decoded, filename, contentType, nil
	}

	return content, filename, contentType, nil
}

// decodeBase64Content decodes base64 encoded content
func (h *Handler) decodeBase64Content(encoded []byte) ([]byte, error) {
	// Try standard base64 decoding first
	decoded, err := base64.StdEncoding.DecodeString(string(encoded))
	if err == nil {
		return decoded, nil
	}

	// Try URL-safe base64 as fallback
	decoded, err = base64.URLEncoding.DecodeString(string(encoded))
	if err == nil {
		return decoded, nil
	}

	// Try raw standard base64 (no padding)
	decoded, err = base64.RawStdEncoding.DecodeString(string(encoded))
	if err == nil {
		return decoded, nil
	}

	// Try raw URL-safe base64 (no padding)
	decoded, err = base64.RawURLEncoding.DecodeString(string(encoded))
	if err != nil {
		return nil, fmt.Errorf("invalid base64 encoding: %v", err)
	}

	return decoded, nil
}

func (h *Handler) readMultipartUpload(c *gin.Context, limit int64) ([]byte, string, string, error) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		return nil, "", "", fmt.Errorf("no file provided")
	}
	defer func() { _ = file.Close() }()

	filename := header.Filename

	// If content is base64 encoded, adjust limit
	effectiveLimit := limit
	if headerEnabled(c, "X-Base64") {
		effectiveLimit = int64(float64(limit) * 1.34)
	}

	if header.Size > 0 && header.Size > effectiveLimit {
		return nil, filename, "", fmt.Errorf("content too large: %d bytes exceeds limit of %d bytes", header.Size, effectiveLimit)
	}

	content, exceeded, err := h.readLimitedContent(file, effectiveLimit)
	if err != nil {
		return nil, filename, "", fmt.Errorf("failed to read file")
	}
	if exceeded {
		return nil, filename, "", fmt.Errorf("content too large: exceeds limit of %d bytes", effectiveLimit)
	}

	contentType := utils.DetectContentType(filename, content)
	if len(content) == 0 {
		return nil, filename, contentType, fmt.Errorf("empty content")
	}
	return content, filename, contentType, nil
}

func (h *Handler) readDirectUpload(c *gin.Context, limit int64) ([]byte, string, string, error) {
	// Adjust limit for base64 overhead if needed
	effectiveLimit := limit
	if headerEnabled(c, "X-Base64") {
		// Base64 increases size by ~33%, plus potential padding
		// Use 1.34x multiplier to account for overhead
		effectiveLimit = int64(float64(limit) * 1.34)
	}

	if contentLength := c.Request.ContentLength; contentLength > 0 && contentLength > effectiveLimit {
		return nil, "", "", fmt.Errorf("content too large: %d bytes exceeds limit of %d bytes", contentLength, effectiveLimit)
	}

	content, exceeded, err := h.readLimitedContent(c.Request.Body, effectiveLimit)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to read content")
	}
	if exceeded {
		return nil, "", "", fmt.Errorf("content too large: exceeds limit of %d bytes", effectiveLimit)
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

func (h *Handler) readLimitedContent(r io.Reader, limit int64) ([]byte, bool, error) {
	if r == nil {
		return nil, false, fmt.Errorf("nil reader")
	}

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

	// Determine if burn-after-read: check X-Burn header first, then fall back to route path
	burnAfterRead := false
	// Support header-presence semantics: X-Burn enables burn unless explicitly disabled
	if headerEnabled(c, "X-Burn") {
		burnAfterRead = true
	} else {
		// Fall back to route-based detection for backward compatibility
		burnAfterRead = strings.HasSuffix(c.FullPath(), "/burn/")
	}

	req := services.CreatePasteRequest{
		Content:       content,
		Filename:      filename,
		ContentType:   contentType,
		BurnAfterRead: burnAfterRead,
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

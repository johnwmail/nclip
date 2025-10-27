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
	"github.com/johnwmail/nclip/models"
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

// dataDir returns the configured data directory. LoadConfig should populate
// Config.DataDir (from flags or environment). We avoid reading the env here
// so that all resolution is centralized in config.LoadConfig.
// dataDir removed: configuration access should use config.DataDir directly.

// Note: rename-to-temp logic has been removed. Burn-after-read is handled by
// streaming content and then deleting the paste from the store for both
// filesystem and S3 backends.

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

// isCli detects if the request is from a CLI tool. It checks the User-Agent
// and also considers the Accept header to avoid misclassifying browsers.
func (h *Handler) isCli(c *gin.Context) bool {
	userAgent := strings.ToLower(c.Request.Header.Get("User-Agent"))
	acceptHeader := c.Request.Header.Get("Accept")

	// If the client explicitly accepts HTML, treat it as a browser.
	if strings.Contains(acceptHeader, "text/html") {
		return false
	}

	// Specific checks for common CLI tools
	cliTools := []string{"curl", "wget", "powershell"}
	for _, tool := range cliTools {
		if strings.Contains(userAgent, tool) {
			return true
		}
	}

	return false
}

// Note: filesystem-specific helper removed; handlers use the same stream-then-delete
// logic for all stores.

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
		log.Printf("[ERROR] View: %v", err)
		h.renderNotFound(c, "Paste not found or deleted")
		return
	}

	// Early strict size check: ask the store for the existence and size of
	// the content. This uses a store-specific stat (filesystem: os.Stat,
	// S3: HeadObject) so it works for either backend without requiring a
	// local dataDir.
	if exists, actualSize, serr := h.store.StatContent(slug); serr == nil && exists {
		if actualSize != paste.Size {
			log.Printf("[ERROR] View: early size mismatch for slug %s: metadata=%d actual=%d", slug, paste.Size, actualSize)
			if h.isCli(c) {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "size_mismatch"})
			} else {
				c.HTML(http.StatusInternalServerError, "view.html", gin.H{
					"Title":      "NCLIP - Error",
					"Error":      "Size mismatch",
					"Version":    h.config.Version,
					"BuildTime":  h.config.BuildTime,
					"CommitHash": h.config.CommitHash,
					"BaseURL":    h.getBaseURL(c),
				})
			}
			return
		}
	}

	// Increment read count
	if err := h.service.IncrementReadCount(slug); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to increment read count for %s: %v\n", slug, err)
	}

	// If burn-after-read, handle specially for CLI and browser clients so the
	// paste is removed on first access.
	if paste.BurnAfterRead {
		if h.isCli(c) {
			h.viewCLI(c, slug, paste)
			return
		}
		// Browser burn handling: render content or preview and remove paste.
		h.viewBrowserBurn(c, slug, paste)
		return
	}

	// CLI clients: full content and streaming (non-burn) or browser clients: render
	if h.isCli(c) {
		h.viewCLI(c, slug, paste)
		return
	}
	h.viewBrowser(c, slug, paste)
}

// viewBrowserBurn handles burn-after-read for browser clients: it should
// provide the same UX as viewBrowser (full vs preview) but ensure the paste
// is deleted on first access. It uses store-provided burn preview/stream
// when available.
// NOTE: Size verification is performed in View() before calling this function.
func (h *Handler) viewBrowserBurn(c *gin.Context, slug string, paste *models.Paste) {
	// For small content, render full
	if paste.Size <= h.config.MaxRenderSize {
		// Read full content, delete paste, render
		full, err := h.service.GetPasteContent(slug)
		if err != nil {
			h.renderNotFound(c, "Paste not available or deleted")
			return
		}
		// No size check needed - already verified in View()
		if err := h.service.DeletePaste(slug); err != nil {
			log.Printf("[ERROR] viewBrowserBurn: failed to delete burn paste %s: %v", slug, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete burn-after-read paste"})
			return
		}
		c.HTML(http.StatusOK, "view.html", gin.H{"Title": fmt.Sprintf("NCLIP - Paste %s", paste.ID), "Paste": paste, "IsText": utils.IsTextContent(paste.ContentType), "IsPreview": false, "Content": string(full), "Version": h.config.Version, "BuildTime": h.config.BuildTime, "CommitHash": h.config.CommitHash, "BaseURL": h.getBaseURL(c)})
		return
	}

	// Large content: reuse loadPreviewContent path which already handles burn
	// semantics for preview-sized reads.
	preview, err := h.loadPreviewContent(c, slug, paste)
	if err != nil {
		return
	}
	c.HTML(http.StatusOK, "view.html", gin.H{"Title": fmt.Sprintf("NCLIP - Paste %s", paste.ID), "Paste": paste, "IsText": utils.IsTextContent(paste.ContentType), "IsPreview": true, "Content": string(preview), "Version": h.config.Version, "BuildTime": h.config.BuildTime, "CommitHash": h.config.CommitHash, "BaseURL": h.getBaseURL(c)})
}

// viewCLI handles CLI (curl/wget/powershell) clients; streams full content or temp file for burn-after-read
// NOTE: Size verification is performed in View() before calling this function.
func (h *Handler) viewCLI(c *gin.Context, slug string, paste *models.Paste) {
	content, err := h.service.GetPasteContent(slug)
	if err != nil {
		log.Printf("[ERROR] View CLI: content not found or deleted for slug %s: %v", slug, err)
		h.renderNotFound(c, "Paste not available or deleted")
		return
	}

	if paste.BurnAfterRead {
		// Delete paste before streaming so subsequent reads return 404
		if err := h.service.DeletePaste(slug); err != nil {
			log.Printf("[ERROR] View CLI: failed to delete burn-after-read paste %s before streaming: %v", slug, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete burn-after-read paste"})
			return
		}
		c.Header("Content-Type", paste.ContentType)
		c.Header("Content-Length", fmt.Sprintf("%d", paste.Size))
		_, _ = c.Writer.Write(content)
		return
	}

	// Non-burn path: serve content normally
	c.Header("Content-Type", paste.ContentType)
	c.Header("Content-Length", fmt.Sprintf("%d", paste.Size))
	c.Data(http.StatusOK, paste.ContentType, content)
}

// viewBrowser handles HTML view rendering; uses MaxRenderSize to determine full vs preview
func (h *Handler) viewBrowser(c *gin.Context, slug string, paste *models.Paste) {
	var content []byte
	isPreview := false

	// If size <= MaxRenderSize render full, otherwise show preview
	if paste.Size <= h.config.MaxRenderSize {
		full, err := h.loadFullContent(slug, paste)
		if err != nil {
			// loadFullContent already logged and rendered appropriate response
			return
		}
		content = full
	} else {
		isPreview = true
		preview, err := h.loadPreviewContent(c, slug, paste)
		if err != nil {
			// loadPreviewContent has already handled rendering/logging
			return
		}
		content = preview
	}

	c.HTML(http.StatusOK, "view.html", gin.H{
		"Title":      fmt.Sprintf("NCLIP - Paste %s", paste.ID),
		"Paste":      paste,
		"IsText":     utils.IsTextContent(paste.ContentType),
		"IsPreview":  isPreview,
		"Content":    string(content),
		"Version":    h.config.Version,
		"BuildTime":  h.config.BuildTime,
		"CommitHash": h.config.CommitHash,
		"BaseURL":    h.getBaseURL(c),
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

	// Early strict size check for Raw: enforce same size_mismatch behavior as View
	if exists, actualSize, serr := h.store.StatContent(slug); serr == nil && exists {
		paste, _ := h.service.GetPaste(slug)
		if paste != nil && actualSize != paste.Size {
			log.Printf("[ERROR] Raw: early size mismatch for slug %s: metadata=%d actual=%d", slug, paste.Size, actualSize)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "size_mismatch"})
			return
		}
	}

	// Increment read count
	if err := h.service.IncrementReadCount(slug); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to increment read count for %s: %v\n", slug, err)
	}

	// Defer content load until after burn-after-read branch to avoid
	// performing unnecessary reads.

	// If burn-after-read, delete the paste so subsequent accesses return 404.
	// Serve the content for this request (first read) then delete the stored data.
	if paste.BurnAfterRead {
		if ok := h.handleRawBurn(c, slug, paste); !ok {
			return
		}
	}
	// Non-burn path: load content now and validate size before serving
	content, cerr := h.service.GetPasteContent(slug)
	if cerr != nil {
		log.Printf("[ERROR] Raw: content not found or deleted for slug %s: %v", slug, cerr)
		c.JSON(http.StatusNotFound, gin.H{"error": "Paste content not found or deleted"})
		return
	}
	// NOTE: early size verification is performed in View(); do not do late checks here.
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

// loadFullContent loads the entire content for small pastes and performs
// strict size checking. It returns the content or an error; on error the
// appropriate response is written to the context by the caller.
func (h *Handler) loadFullContent(slug string, paste *models.Paste) ([]byte, error) {
	full, err := h.service.GetPasteContent(slug)
	if err != nil {
		log.Printf("[ERROR] View Browser: content not found or deleted for slug %s: %v", slug, err)
		return nil, err
	}
	// NOTE: early size verification is performed in View(); return the full content.
	return full, nil
}

// loadPreviewContent loads a preview (up to MaxRenderSize) for large pastes.
// It handles burn-after-read temp rename branches, preview reads and returns
// the preview bytes or an error. The caller must render the response when an
// error is returned.
func (h *Handler) loadPreviewContent(c *gin.Context, slug string, paste *models.Paste) ([]byte, error) {
	if !utils.IsTextContent(paste.ContentType) {
		return []byte(""), nil
	}
	if paste.BurnAfterRead {
		// Read up to MaxRenderSize, verify full size via StatContent when possible,
		// delete the paste, and return the prefix for preview rendering.
		prefix, err := h.store.GetContentPrefix(slug, h.config.MaxRenderSize)
		if err != nil {
			log.Printf("[ERROR] View Browser: failed to read preview from store for %s: %v", slug, err)
			h.renderNotFound(c, "Paste not available or deleted")
			return nil, err
		}
		// Verify size matches metadata before deleting, if we can stat the content.
		// No late verification of size here; delete paste and return preview.
		if err := h.service.DeletePaste(slug); err != nil {
			log.Printf("[ERROR] View Browser: failed to delete burn-after-read paste %s during preview: %v", slug, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete burn-after-read paste"})
			return nil, err
		}
		return prefix, nil
	}

	prefix, err := h.store.GetContentPrefix(slug, h.config.MaxRenderSize)
	if err != nil {
		log.Printf("[ERROR] View Browser: failed to read preview from store for %s: %v", slug, err)
		h.renderNotFound(c, "Paste not available or deleted")
		return nil, err
	}
	return prefix, nil
}

// handleRawBurn performs the burn-after-read flow for Raw: it moves files to
// temporary burn paths, validates size, deletes metadata, streams the file
// to the response, and cleans up. It returns true on success (streamed and
// returned) or false if the caller should stop processing.
func (h *Handler) handleRawBurn(c *gin.Context, slug string, paste *models.Paste) bool {
	// Unified handler-level burn: read full content, verify size, delete paste, then stream the bytes.
	// Read full content, verify size, delete the paste, then stream the bytes.
	content, err := h.service.GetPasteContent(slug)
	if err != nil {
		log.Printf("[ERROR] Raw: content not found or deleted for slug %s: %v", slug, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Paste not found or deleted"})
		return false
	}
	// No late size mismatch checks; proceed to delete and stream.
	if err := h.service.DeletePaste(slug); err != nil {
		log.Printf("[ERROR] Raw: failed to delete burn-after-read paste %s before streaming: %v", slug, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete burn-after-read paste"})
		return false
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
	_, _ = c.Writer.Write(content)
	return true
}

// renderNotFound sends a consistent 404 response. CLI/API clients receive JSON,
// while browser clients receive the HTML view with a friendly message.
func (h *Handler) renderNotFound(c *gin.Context, message string) {
	if h.isCli(c) {
		c.JSON(http.StatusNotFound, gin.H{"error": message})
	} else {
		c.Header("Content-Type", "text/html; charset=utf-8")
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

package retrieval

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// tempRenameInDataDir attempts to atomically move the paste content and metadata
// into temporary burn files inside the same data directory. It returns the
// temp content and meta paths. If rename fails, it returns an error.
func (h *Handler) tempRenameInDataDir(slug string) (tmpContentPath, tmpMetaPath string, err error) {
	dataDir := os.Getenv("NCLIP_DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	// Ensure dataDir exists, return error if not exists
	if _, statErr := os.Stat(dataDir); os.IsNotExist(statErr) {
		return "", "", statErr
	}

	contentPath := filepath.Join(dataDir, slug)
	metaPath := filepath.Join(dataDir, slug+".json")
	ts := time.Now().UnixNano()
	tmpContent := filepath.Join(dataDir, fmt.Sprintf("%s.burn.%d", slug, ts))
	tmpMeta := filepath.Join(dataDir, fmt.Sprintf("%s.burn.%d.json", slug, ts))

	// content must exist to proceed
	if _, statErr := os.Stat(contentPath); statErr != nil {
		return "", "", fmt.Errorf("content not found: %w", statErr)
	}

	// try to rename content
	if err := os.Rename(contentPath, tmpContent); err != nil {
		return "", "", fmt.Errorf("rename content failed: %w", err)
	}

	// If metadata exists, attempt to rename it too. If it fails, revert content rename.
	if _, statErr := os.Stat(metaPath); statErr == nil {
		if err := os.Rename(metaPath, tmpMeta); err != nil {
			// revert content
			_ = os.Rename(tmpContent, contentPath)
			return "", "", fmt.Errorf("rename meta failed: %w", err)
		}
	}

	return tmpContent, tmpMeta, nil
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
		log.Printf("[ERROR] View: %v", err)
		h.renderNotFound(c, "Paste not found or deleted")
		return
	}

	// Increment read count
	if err := h.service.IncrementReadCount(slug); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to increment read count for %s: %v\n", slug, err)
	}

	// CLI clients: full content and streaming from temp file for burn-after-read
	if h.isCli(c) {
		h.viewCLI(c, slug, paste)
		return
	}

	// Browser clients: render full or preview depending on threshold
	h.viewBrowser(c, slug, paste)
}

// viewCLI handles CLI (curl/wget/powershell) clients; streams full content or temp file for burn-after-read
func (h *Handler) viewCLI(c *gin.Context, slug string, paste *models.Paste) {
	content, err := h.service.GetPasteContent(slug)
	if err != nil {
		log.Printf("[ERROR] View CLI: content not found or deleted for slug %s: %v", slug, err)
		h.renderNotFound(c, "Paste not available or deleted")
		return
	}
	if paste.BurnAfterRead {
		tmpContentPath, tmpMetaPath, err := h.tempRenameInDataDir(slug)
		if err != nil {
			log.Printf("[ERROR] View CLI: failed to prepare burn-after-read: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare burn-after-read content"})
			return
		}
		if err := h.service.DeletePaste(slug); err != nil {
			log.Printf("[ERROR] View CLI: failed to delete burn-after-read paste %s after temping: %v", slug, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete burn-after-read paste"})
			return
		}
		f, err := os.Open(tmpContentPath)
		if err != nil {
			log.Printf("[ERROR] View CLI: failed to open temp file for streaming: %v", err)
			_ = os.Remove(tmpContentPath)
			_ = os.Remove(tmpMetaPath)
			h.renderNotFound(c, "Paste not available or deleted")
			return
		}
		defer func() {
			if cerr := f.Close(); cerr != nil {
				log.Printf("[WARN] View CLI: failed to close temp file: %v", cerr)
			}
			_ = os.Remove(tmpContentPath)
			_ = os.Remove(tmpMetaPath)
		}()
		c.Header("Content-Type", paste.ContentType)
		c.Header("Content-Length", fmt.Sprintf("%d", paste.Size))
		_, _ = io.Copy(c.Writer, f)
		return
	}
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
		full, err := h.service.GetPasteContent(slug)
		if err != nil {
			log.Printf("[ERROR] View Browser: content not found or deleted for slug %s: %v", slug, err)
			h.renderNotFound(c, "Paste not available or deleted")
			return
		}
		content = full
	} else {
		isPreview = true
		if utils.IsTextContent(paste.ContentType) {
			// preview path
			if paste.BurnAfterRead {
				tmpContentPath, tmpMetaPath, err := h.tempRenameInDataDir(slug)
				if err != nil {
					log.Printf("[ERROR] View Browser: failed to prepare burn-after-read: %v", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare burn-after-read content"})
					return
				}
				if err := h.service.DeletePaste(slug); err != nil {
					log.Printf("[ERROR] View Browser: failed to delete burn-after-read paste %s after temping: %v", slug, err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete burn-after-read paste"})
					return
				}
				f, err := os.Open(tmpContentPath)
				if err != nil {
					log.Printf("[ERROR] View Browser: failed to open temp file for preview: %v", err)
					_ = os.Remove(tmpContentPath)
					_ = os.Remove(tmpMetaPath)
					h.renderNotFound(c, "Paste not available or deleted")
					return
				}
				preview := make([]byte, h.config.MaxRenderSize)
				n, err := io.ReadFull(f, preview)
				if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
					if cerr := f.Close(); cerr != nil {
						log.Printf("[WARN] View Browser: failed to close temp file after read error: %v", cerr)
					}
					_ = os.Remove(tmpContentPath)
					_ = os.Remove(tmpMetaPath)
					log.Printf("[ERROR] View Browser: failed to read preview from temp file: %v", err)
					h.renderNotFound(c, "Paste not available or deleted")
					return
				}
				if cerr := f.Close(); cerr != nil {
					log.Printf("[WARN] View Browser: failed to close temp file: %v", cerr)
				}
				_ = os.Remove(tmpContentPath)
				_ = os.Remove(tmpMetaPath)
				content = preview[:n]
			} else {
				prefix, err := h.store.GetContentPrefix(slug, h.config.MaxRenderSize)
				if err != nil {
					log.Printf("[ERROR] View Browser: failed to read preview from store for %s: %v", slug, err)
					h.renderNotFound(c, "Paste not available or deleted")
					return
				}
				content = prefix
			}
		} else {
			content = []byte("")
		}
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
		tmpContentPath, tmpMetaPath, err := h.tempRenameInDataDir(slug)
		if err != nil {
			log.Printf("[ERROR] Raw: failed to prepare burn-after-read: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare burn-after-read content"})
			return
		}

		// If Delete fails, return 500.
		if err := h.service.DeletePaste(slug); err != nil {
			log.Printf("[ERROR] Raw: failed to delete burn-after-read paste %s after temping: %v", slug, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete burn-after-read paste"})
			return
		}

		f, err := os.Open(tmpContentPath)
		if err != nil {
			log.Printf("[ERROR] Raw: failed to open temp file for streaming: %v", err)
			_ = os.Remove(tmpContentPath)
			_ = os.Remove(tmpMetaPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Paste not available or deleted"})
			return
		}
		defer func() {
			if cerr := f.Close(); cerr != nil {
				log.Printf("[WARN] Raw: failed to close temp file: %v", cerr)
			}
			_ = os.Remove(tmpContentPath)
			_ = os.Remove(tmpMetaPath)
		}()

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
		_, _ = io.Copy(c.Writer, f)
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

package retrieval

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/config"
	"github.com/johnwmail/nclip/internal/services"
	"github.com/johnwmail/nclip/models"
	"github.com/johnwmail/nclip/storage"
)

// Test that when metadata Size != actual content size the handler returns size_mismatch (500)
func TestView_SizeMismatch_CLI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataDir := t.TempDir()
	cfg := &config.Config{MaxRenderSize: 1024, DataDir: dataDir}
	fs, err := storage.NewFilesystemStore(dataDir)
	if err != nil {
		t.Fatalf("failed to create filesystem store: %v", err)
	}
	store := fs
	svc := services.NewPasteService(store, cfg)
	rh := NewHandler(svc, store, cfg)

	// Create paste where metadata Size is wrong (larger than actual content)
	content := []byte("short")
	id := "HJK23"
	paste := &models.Paste{
		ID:            id,
		CreatedAt:     time.Now(),
		Size:          int64(len(content) + 10),
		ContentType:   "text/plain",
		Content:       content,
		BurnAfterRead: false,
	}
	if err := store.StoreContent(paste.ID, paste.Content); err != nil {
		t.Fatalf("failed to store content: %v", err)
	}
	if err := store.Store(paste); err != nil {
		t.Fatalf("failed to store paste metadata: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/HJK23", nil)
	req.Header.Set("User-Agent", "curl/7.64.1")
	c.Request = req
	c.Params = gin.Params{{Key: "slug", Value: id}}

	rh.View(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for size mismatch, got %d, body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "size_mismatch") {
		t.Fatalf("expected size_mismatch in response body, got: %s", w.Body.String())
	}
}

func TestRaw_SizeMismatch_NonBurn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataDir := t.TempDir()
	cfg := &config.Config{MaxRenderSize: 1024, DataDir: dataDir}
	fs, err := storage.NewFilesystemStore(dataDir)
	if err != nil {
		t.Fatalf("failed to create filesystem store: %v", err)
	}
	store := fs
	svc := services.NewPasteService(store, cfg)
	rh := NewHandler(svc, store, cfg)

	content := []byte("abc")
	id2 := "LMN34"
	paste := &models.Paste{ID: id2, CreatedAt: time.Now(), Size: int64(len(content) + 5), ContentType: "text/plain", Content: content, BurnAfterRead: false}
	if err := store.StoreContent(paste.ID, paste.Content); err != nil {
		t.Fatalf("failed to store content: %v", err)
	}
	if err := store.Store(paste); err != nil {
		t.Fatalf("failed to store paste metadata: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/raw/LMN34", nil)
	req.Header.Set("User-Agent", "curl/7.64.1")
	c.Request = req
	c.Params = gin.Params{{Key: "slug", Value: id2}}

	rh.Raw(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for size mismatch on raw, got %d, body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "size_mismatch") {
		t.Fatalf("expected size_mismatch in raw response body, got: %s", w.Body.String())
	}
}

package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/config"
	"github.com/johnwmail/nclip/handlers/retrieval"

	"github.com/johnwmail/nclip/internal/services"
	"github.com/johnwmail/nclip/models"
)

// Test that small content is rendered fully in HTML view
func TestViewSmallRendersFull(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{MaxRenderSize: 1024}
	store := NewMockStore(cfg.DataDir)
	defer cleanupTestData(store.dataDir)
	svc := services.NewPasteService(store, cfg)
	rh := retrieval.NewHandler(svc, store, cfg)

	// Create small paste
	paste := &models.Paste{ID: "SMALLA", CreatedAt: time.Now(), Size: 11, ContentType: "text/plain", Content: []byte("hello world"), BurnAfterRead: false}
	store.Store(paste)

	// Call handler directly using a test context to avoid router matching issues
	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)
	engine.LoadHTMLGlob("static/*.html")
	c.Request = httptest.NewRequest("GET", "/SMALLA", nil)
	c.Params = gin.Params{{Key: "slug", Value: "SMALLA"}}
	rh.View(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "hello world") {
		t.Fatalf("expected full content in HTML view, got body: %s", body)
	}
}

// Test that large content shows a preview (not full content) in HTML view
func TestViewLargeShowsPreview(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{MaxRenderSize: 10}
	store := NewMockStore(cfg.DataDir)
	defer cleanupTestData(store.dataDir)
	svc := services.NewPasteService(store, cfg)
	rh := retrieval.NewHandler(svc, store, cfg)

	// Create large paste
	long := strings.Repeat("A", 50)
	paste := &models.Paste{ID: "LARGEA", CreatedAt: time.Now(), Size: int64(len(long)), ContentType: "text/plain", Content: []byte(long), BurnAfterRead: false}
	store.Store(paste)

	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)
	engine.LoadHTMLGlob("static/*.html")
	c.Request = httptest.NewRequest("GET", "/LARGEA", nil)
	c.Params = gin.Params{{Key: "slug", Value: "LARGEA"}}
	rh.View(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, long) {
		t.Fatalf("expected preview, not full content")
	}
	// preview should contain MaxRenderSize bytes of 'A'
	if !strings.Contains(body, strings.Repeat("A", int(cfg.MaxRenderSize))) {
		t.Fatalf("expected preview substring in body")
	}
}

// Integration-style test: burn-after-read preview reads prefix from temp file and deletes it
func TestBurnAfterReadPreviewDeletesTemp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{MaxRenderSize: 10}
	store := NewMockStore(cfg.DataDir)
	defer cleanupTestData(store.dataDir)
	svc := services.NewPasteService(store, cfg)
	rh := retrieval.NewHandler(svc, store, cfg)

	// Create burn paste
	content := strings.Repeat("B", 32)
	paste := &models.Paste{ID: "BURN2", CreatedAt: time.Now(), Size: int64(len(content)), ContentType: "text/plain", Content: []byte(content), BurnAfterRead: true}
	store.Store(paste)

	// ensure data dir exists and the file is present (MockStore writes file)
	dataDir := cfg.DataDir
	if dataDir == "" {
		dataDir = "./data"
	}
	// sanity check file exists
	if _, err := os.Stat(filepath.Join(dataDir, "BURN2")); err != nil {
		t.Fatalf("expected content file on disk before request: %v", err)
	}

	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)
	engine.LoadHTMLGlob("static/*.html")
	c.Request = httptest.NewRequest("GET", "/BURN2", nil)
	c.Params = gin.Params{{Key: "slug", Value: "BURN2"}}
	rh.View(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	// after request, original file should be gone (moved then deleted by handler)
	if _, err := os.Stat(filepath.Join(dataDir, "BURN2")); !os.IsNotExist(err) {
		t.Fatalf("expected original content removed after burn, stat err: %v", err)
	}
}

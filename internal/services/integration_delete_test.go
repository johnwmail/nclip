package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/johnwmail/nclip/config"
	"github.com/johnwmail/nclip/models"
	"github.com/johnwmail/nclip/storage"
)

// TestIntegrationDeleteExpired verifies that an expired paste is physically removed
// from the configured data directory. If the environment variable NCLIP_DATA_DIR is
// set, the test will use that directory (useful for manual/CI integration). If not
// set, the test uses a temporary directory and cleans it up.
func TestIntegrationDeleteExpired(t *testing.T) {
	// Prefer an explicit temp dir for integration test so we don't depend on
	// environment and can run in CI reliably.
	cfg := config.LoadConfig()
	dir := t.TempDir()
	// Override cfg DataDir for this test to ensure service and store use the temp dir.
	cfg.DataDir = dir

	store, err := storage.NewFilesystemStore(dir)
	if err != nil {
		t.Fatalf("failed to create filesystem store: %v", err)
	}

	svc := NewPasteService(store, cfg)

	slug := "integ-expired-" + time.Now().Format("20060102150405")
	expires := time.Now().Add(-1 * time.Hour)
	paste := &models.Paste{
		ID:          slug,
		CreatedAt:   time.Now().Add(-2 * time.Hour),
		ExpiresAt:   &expires,
		Size:        1,
		ContentType: "text/plain",
	}

	if err := store.StoreContent(slug, []byte("x")); err != nil {
		t.Fatalf("failed to store content: %v", err)
	}
	if err := store.Store(paste); err != nil {
		t.Fatalf("failed to store metadata: %v", err)
	}

	// Trigger the service path that should delete expired paste on access
	_, err = svc.GetPaste(slug)
	if err == nil {
		t.Fatalf("expected error when retrieving expired paste")
	}

	// Verify files do not exist in the configured directory
	metaPath := filepath.Join(dir, slug+".json")
	if _, err := os.Stat(metaPath); !os.IsNotExist(err) {
		t.Fatalf("expected metadata file removed from %s, still exists", metaPath)
	}
	contentPath := filepath.Join(dir, slug)
	if _, err := os.Stat(contentPath); !os.IsNotExist(err) {
		t.Fatalf("expected content file removed from %s, still exists", contentPath)
	}
}

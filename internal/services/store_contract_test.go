package services

import (
	"os"
	"testing"
	"time"

	"github.com/johnwmail/nclip/config"
	"github.com/johnwmail/nclip/models"
	"github.com/johnwmail/nclip/storage"
)

// storeFactory creates a storage.PasteStore for testing and returns a cleanup func
type storeFactory func(t *testing.T) (storage.PasteStore, func())

func fsFactory(t *testing.T) (storage.PasteStore, func()) {
	dir, err := os.MkdirTemp("", "nclip-store-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	store, err := storage.NewFilesystemStore(dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		t.Fatalf("failed to create filesystem store: %v", err)
	}
	cleanup := func() { os.RemoveAll(dir) }
	return store, cleanup
}

func TestStoreContract_DeleteExpiredOnAccess(t *testing.T) {
	store, cleanup := fsFactory(t)
	defer cleanup()

	svc := NewPasteService(store, &config.Config{})

	// create expired paste
	slug := "contract-expired"
	expires := time.Now().Add(-1 * time.Hour)
	paste := &models.Paste{
		ID:          slug,
		CreatedAt:   time.Now().Add(-2 * time.Hour),
		ExpiresAt:   &expires,
		Size:        1,
		ContentType: "text/plain",
	}

	if err := store.StoreContent(slug, []byte("x")); err != nil {
		t.Fatalf("store content failed: %v", err)
	}
	if err := store.Store(paste); err != nil {
		t.Fatalf("store metadata failed: %v", err)
	}

	// Call GetPaste; it should delete expired paste and return error
	_, err := svc.GetPaste(slug)
	if err == nil {
		t.Fatalf("expected error retrieving expired paste, got nil")
	}

	// Confirm metadata and content removed (FilesystemStore specifics)
	// We can't access the store internals here, but FilesystemStore stores files in a directory; attempt to stat
	// Attempt to guess the underlying dir by asking the FS implementation is not exposed; instead use Exists()
	exists, _ := store.Exists(slug)
	if exists {
		t.Fatalf("expected store to not have slug after expired access")
	}
}

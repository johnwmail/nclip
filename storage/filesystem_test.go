package storage

import (
	"os"
	"testing"

	"github.com/johnwmail/nclip/models"
)

func TestNewFilesystemStore_Defaults(t *testing.T) {
	os.Setenv("NCLIP_DATA_DIR", "./testdata")
	os.Setenv("NCLIP_S3_BUCKET", "")
	store, err := NewFilesystemStore()
	if err != nil {
		t.Fatalf("NewFilesystemStore failed: %v", err)
	}
	if store.dataDir != "./testdata" {
		t.Errorf("expected dataDir ./testdata, got %s", store.dataDir)
	}
	if store.useS3 {
		t.Errorf("expected useS3 false, got true")
	}
}

func TestFilesystemStore_StoreAndGet_LocalFS(t *testing.T) {
	os.Setenv("NCLIP_DATA_DIR", "./testdata")
	os.Setenv("NCLIP_S3_BUCKET", "")
	store, err := NewFilesystemStore()
	if err != nil {
		t.Fatalf("NewFilesystemStore failed: %v", err)
	}
	paste := &models.Paste{ID: "testfs", Size: 123}
	err = store.Store(paste)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	got, err := store.Get("testfs")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != "testfs" {
		t.Errorf("expected ID testfs, got %s", got.ID)
	}
	_ = store.Delete("testfs")
}

func TestFilesystemStore_Delete_LocalFS(t *testing.T) {
	os.Setenv("NCLIP_DATA_DIR", "./testdata")
	os.Setenv("NCLIP_S3_BUCKET", "")
	store, err := NewFilesystemStore()
	if err != nil {
		t.Fatalf("NewFilesystemStore failed: %v", err)
	}
	paste := &models.Paste{ID: "todel", Size: 1}
	_ = store.Store(paste)
	err = store.Delete("todel")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}
}

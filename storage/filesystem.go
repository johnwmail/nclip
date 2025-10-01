package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/johnwmail/nclip/models"
	"github.com/johnwmail/nclip/utils"
)

// isSafeFilename checks that the provided filename/id is a single name (not a path)
func isSafeFilename(name string) bool {
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return false
	}
	return true
}

// FilesystemStore stores paste metadata and content on the local filesystem.
type FilesystemStore struct {
	dataDir    string
	bufferSize int
	mu         sync.Mutex
}

// NewFilesystemStore creates a FilesystemStore for the given data directory.
// If dataDir is empty it defaults to "./data". The directory is created if it does not exist.
func NewFilesystemStore(dataDir string) (*FilesystemStore, error) {
	if dataDir == "" {
		dataDir = "./data"
	}
	// Check if the dataDir not exists and create it with logging
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		log.Printf("[INFO] Creating data directory: %s", dataDir)
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create data directory %s: %w", dataDir, err)
		}
	}
	return &FilesystemStore{
		dataDir:    dataDir,
		bufferSize: 4096,
	}, nil
}

// Store saves the paste metadata (JSON) to local filesystem
func (fs *FilesystemStore) Store(paste *models.Paste) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	metaData, err := json.MarshalIndent(paste, "", "  ")
	if err != nil {
		log.Printf("[ERROR] FS Store: failed to marshal metadata for %s: %v", paste.ID, err)
		return err
	}
	metaPath := filepath.Join(fs.dataDir, paste.ID+".json")
	if err := os.WriteFile(metaPath, metaData, 0o644); err != nil {
		log.Printf("[ERROR] FS Store: failed to write metadata for %s: %v", paste.ID, err)
		return err
	}
	return nil
}

func (fs *FilesystemStore) Get(id string) (*models.Paste, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	metaPath := filepath.Join(fs.dataDir, id+".json")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[ERROR] FS Get: failed to read metadata for %s: %v", id, err)
		}
		return nil, err
	}
	var paste models.Paste
	if err := json.Unmarshal(metaData, &paste); err != nil {
		log.Printf("[ERROR] Get: failed to unmarshal metadata for %s: %v", id, err)
		return nil, err
	}
	if paste.IsExpired() {
		log.Printf("[INFO] FS Get: paste %s is expired", id)
		if !isSafeFilename(id) {
			log.Printf("[ERROR] FS Get: unsafe id: %q", id)
			return nil, fmt.Errorf("invalid paste id")
		}
		return nil, os.ErrNotExist
	}
	return &paste, nil
}

func (fs *FilesystemStore) Exists(id string) (bool, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	metaPath := filepath.Join(fs.dataDir, id+".json")
	_, err := os.Stat(metaPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		log.Printf("[ERROR] FS Exists: failed to stat metadata for %s: %v", id, err)
		if !isSafeFilename(id) {
			log.Printf("[ERROR] FS Exists: unsafe id: %q", id)
			return false, fmt.Errorf("invalid paste id")
		}
		return false, err
	}
	return true, nil
}

func (fs *FilesystemStore) Delete(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	contentPath := filepath.Join(fs.dataDir, id)
	metaPath := filepath.Join(fs.dataDir, id+".json")
	if !isSafeFilename(id) {
		log.Printf("[ERROR] FS IncrementReadCount: unsafe id: %q", id)
		return fmt.Errorf("invalid paste id")
	}
	_ = os.Remove(contentPath)
	_ = os.Remove(metaPath)
	return nil
}

func (fs *FilesystemStore) IncrementReadCount(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	metaPath := filepath.Join(fs.dataDir, id+".json")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		log.Printf("[ERROR] FS IncrementReadCount: failed to read metadata for %s: %v", id, err)
		return err
	}
	var paste models.Paste
	if err := json.Unmarshal(metaData, &paste); err != nil {
		log.Printf("[ERROR] FS IncrementReadCount: failed to unmarshal metadata for %s: %v", id, err)
		return err
	}
	paste.ReadCount++
	newMeta, err := json.MarshalIndent(&paste, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(metaPath, newMeta, 0o644); err != nil {
		log.Printf("[ERROR] FS IncrementReadCount: failed to write metadata for %s: %v", id, err)
		if !isSafeFilename(id) {
			log.Printf("[ERROR] FS StoreContent: unsafe id: %q", id)
			return fmt.Errorf("invalid paste id")
		}
		return err
	}
	return nil
}

func (fs *FilesystemStore) StoreContent(id string, content []byte) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if utils.IsDebugEnabled() {
		log.Printf("[DEBUG] StoreContent: id=%s, content_len=%d, first_bytes=%q", id, len(content), string(content[:min(32, len(content))]))
	}
	if err := os.MkdirAll(fs.dataDir, 0o755); err != nil {
		log.Printf("[ERROR] FS StoreContent: failed to create data directory %s: %v", fs.dataDir, err)
		return err
	}
	contentPath := filepath.Join(fs.dataDir, id)
	if err := os.WriteFile(contentPath, content, 0o644); err != nil {
		log.Printf("[ERROR] FS StoreContent: failed to write content for %s: %v", id, err)
		return err
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (fs *FilesystemStore) GetContent(id string) ([]byte, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	contentPath := filepath.Join(fs.dataDir, id)
	data, err := os.ReadFile(contentPath)
	if err != nil {
		log.Printf("[ERROR] FS GetContent: failed to read content for %s: %v", id, err)
		return nil, err
	}
	return data, nil
}

func (fs *FilesystemStore) Close() error {
	return nil
}

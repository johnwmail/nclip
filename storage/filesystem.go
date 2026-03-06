package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/johnwmail/nclip/models"
	"github.com/johnwmail/nclip/utils"
)

// errUnsafeID is returned when an id contains path traversal characters.
var errUnsafeID = fmt.Errorf("invalid paste id")

// safePath constructs a filesystem path for the given id under baseDir,
// returning an error if the resulting path escapes the base directory.
// This uses filepath.Clean to normalize the path and then verifies it
// stays within baseDir, which is the pattern recognised by CodeQL.
func safePath(baseDir, id string) (string, error) {
	// Reject obvious traversal characters early for clear error messages.
	if strings.Contains(id, "/") || strings.Contains(id, "\\") || strings.Contains(id, "..") {
		log.Printf("[ERROR] FS: unsafe id rejected: %q", id)
		return "", errUnsafeID
	}
	p := filepath.Join(baseDir, id)
	p = filepath.Clean(p)
	// Ensure the cleaned path is still within baseDir.
	if !strings.HasPrefix(p, filepath.Clean(baseDir)+string(os.PathSeparator)) {
		log.Printf("[ERROR] FS: path escapes base dir: %q", p)
		return "", errUnsafeID
	}
	return p, nil
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
	metaPath, err := safePath(fs.dataDir, paste.ID+".json")
	if err != nil {
		return err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	metaData, err := json.MarshalIndent(paste, "", "  ")
	if err != nil {
		log.Printf("[ERROR] FS Store: failed to marshal metadata for %s: %v", paste.ID, err)
		return err
	}
	if err := os.WriteFile(metaPath, metaData, 0o644); err != nil {
		log.Printf("[ERROR] FS Store: failed to write metadata for %s: %v", paste.ID, err)
		return err
	}
	return nil
}

func (fs *FilesystemStore) Get(id string) (*models.Paste, error) {
	metaPath, err := safePath(fs.dataDir, id+".json")
	if err != nil {
		return nil, err
	}
	contentPath, err := safePath(fs.dataDir, id)
	if err != nil {
		return nil, err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		log.Printf("[ERROR] FS Get: failed to read metadata for %s: %v", id, err)
		return nil, err
	}
	var paste models.Paste
	if err := json.Unmarshal(metaData, &paste); err != nil {
		log.Printf("[ERROR] Get: failed to unmarshal metadata for %s: %v", id, err)
		return nil, err
	}
	if paste.IsExpired() {
		log.Printf("[WARN] FS Get: paste %s is expired", id)
		// Delete expired paste files directly (we already hold the mutex) so subsequent accesses are clean
		_ = os.Remove(contentPath)
		if err := os.Remove(metaPath); err != nil {
			log.Printf("[WARN] FS Get: failed to remove expired metadata for %s: %v", id, err)
		}
		return nil, ErrNotFound
	}
	return &paste, nil
}

func (fs *FilesystemStore) Exists(id string) (bool, error) {
	metaPath, err := safePath(fs.dataDir, id+".json")
	if err != nil {
		return false, err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	_, err = os.Stat(metaPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		log.Printf("[ERROR] FS Exists: failed to stat metadata for %s: %v", id, err)
		return false, err
	}
	return true, nil
}

func (fs *FilesystemStore) Delete(id string) error {
	contentPath, err := safePath(fs.dataDir, id)
	if err != nil {
		return err
	}
	metaPath, err := safePath(fs.dataDir, id+".json")
	if err != nil {
		return err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	_ = os.Remove(contentPath)
	_ = os.Remove(metaPath)
	return nil
}

func (fs *FilesystemStore) IncrementReadCount(id string) error {
	metaPath, err := safePath(fs.dataDir, id+".json")
	if err != nil {
		return err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
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
		return err
	}
	return nil
}

func (fs *FilesystemStore) StoreContent(id string, content []byte) error {
	contentPath, err := safePath(fs.dataDir, id)
	if err != nil {
		return err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if utils.IsDebugEnabled() {
		log.Printf("[DEBUG] StoreContent: id=%s, content_len=%d, first_bytes=%q", id, len(content), string(content[:min(32, len(content))]))
	}
	if err := os.MkdirAll(fs.dataDir, 0o755); err != nil {
		log.Printf("[ERROR] FS StoreContent: failed to create data directory %s: %v", fs.dataDir, err)
		return err
	}
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
	contentPath, err := safePath(fs.dataDir, id)
	if err != nil {
		return nil, err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	data, err := os.ReadFile(contentPath)
	if err != nil {
		log.Printf("[ERROR] FS GetContent: failed to read content for %s: %v", id, err)
		return nil, err
	}
	return data, nil
}

// StatContent reports whether content exists on disk and its size.
func (fs *FilesystemStore) StatContent(id string) (bool, int64, error) {
	contentPath, err := safePath(fs.dataDir, id)
	if err != nil {
		return false, 0, err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	st, err := os.Stat(contentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, 0, nil
		}
		log.Printf("[ERROR] FS StatContent: failed to stat content for %s: %v", id, err)
		return false, 0, err
	}
	return true, st.Size(), nil
}

// GetContentPrefix reads up to n bytes from the content file. If the file is
// smaller than n, it returns the full content.
func (fs *FilesystemStore) GetContentPrefix(id string, n int64) ([]byte, error) {
	contentPath, err := safePath(fs.dataDir, id)
	if err != nil {
		return nil, err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	f, err := os.Open(contentPath)
	if err != nil {
		log.Printf("[ERROR] FS GetContentPrefix: failed to open content for %s: %v", id, err)
		return nil, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Printf("[WARN] FS GetContentPrefix: failed to close file for %s: %v", id, cerr)
		}
	}()
	// If n is small enough to allocate, read into buffer
	buf := make([]byte, n)
	read, err := io.ReadFull(f, buf)
	if err != nil {
		if err == io.ErrUnexpectedEOF || err == io.EOF {
			return buf[:read], nil
		}
		if err == io.ErrUnexpectedEOF {
			return buf[:read], nil
		}
		return nil, err
	}
	return buf[:read], nil
}

func (fs *FilesystemStore) Close() error {
	return nil
}

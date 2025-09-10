package storage

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FilesystemStorage implements Storage interface using the filesystem
type FilesystemStorage struct {
	basePath string
	mu       sync.RWMutex
}

// NewFilesystemStorage creates a new filesystem storage backend
func NewFilesystemStorage(basePath string) (*FilesystemStorage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &FilesystemStorage{
		basePath: basePath,
	}, nil
}

// Store saves a paste to the filesystem
func (f *FilesystemStorage) Store(paste *Paste) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Create directory for the paste
	pasteDir := filepath.Join(f.basePath, paste.ID)
	if err := os.MkdirAll(pasteDir, 0755); err != nil {
		return fmt.Errorf("failed to create paste directory: %w", err)
	}

	// Save content to index.txt (for fiche compatibility)
	contentPath := filepath.Join(pasteDir, "index.txt")
	if err := os.WriteFile(contentPath, paste.Content, 0644); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	// Save metadata to metadata.json
	metadataPath := filepath.Join(pasteDir, "metadata.json")
	metadataBytes, err := json.MarshalIndent(paste, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataBytes, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// Get retrieves a paste from the filesystem
func (f *FilesystemStorage) Get(id string) (*Paste, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	pasteDir := filepath.Join(f.basePath, id)

	// Check if paste directory exists
	if _, err := os.Stat(pasteDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("paste not found: %s", id)
	}

	// Read metadata
	metadataPath := filepath.Join(pasteDir, "metadata.json")
	var paste Paste

	if metadataBytes, err := os.ReadFile(metadataPath); err == nil {
		// New format with metadata
		if err := json.Unmarshal(metadataBytes, &paste); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	} else {
		// Legacy format or missing metadata - reconstruct from content
		paste.ID = id
		paste.CreatedAt = f.getCreationTime(pasteDir)
		paste.ContentType = "text/plain"
	}

	// Read content
	contentPath := filepath.Join(pasteDir, "index.txt")
	content, err := os.ReadFile(contentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	paste.Content = content
	paste.Size = int64(len(content))

	// Check if paste has expired
	if paste.ExpiresAt != nil && time.Now().After(*paste.ExpiresAt) {
		return nil, fmt.Errorf("paste has expired: %s", id)
	}

	return &paste, nil
}

// Exists checks if a paste exists
func (f *FilesystemStorage) Exists(id string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	pasteDir := filepath.Join(f.basePath, id)
	contentPath := filepath.Join(pasteDir, "index.txt")

	_, err := os.Stat(contentPath)
	return err == nil
}

// Delete removes a paste
func (f *FilesystemStorage) Delete(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	pasteDir := filepath.Join(f.basePath, id)
	return os.RemoveAll(pasteDir)
}

// List returns a list of paste IDs
func (f *FilesystemStorage) List(limit int) ([]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	entries, err := os.ReadDir(f.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read base directory: %w", err)
	}

	var ids []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if it contains index.txt
			contentPath := filepath.Join(f.basePath, entry.Name(), "index.txt")
			if _, err := os.Stat(contentPath); err == nil {
				ids = append(ids, entry.Name())

				if limit > 0 && len(ids) >= limit {
					break
				}
			}
		}
	}

	return ids, nil
}

// Stats returns storage statistics
func (f *FilesystemStorage) Stats() (*Stats, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	stats := &Stats{}

	err := filepath.WalkDir(f.basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Count index.txt files as pastes
		if strings.HasSuffix(path, "index.txt") {
			stats.TotalPastes++

			// Get file size
			if info, err := d.Info(); err == nil {
				stats.TotalSize += info.Size()
			}
		}

		return nil
	})

	return stats, err
}

// Cleanup removes expired pastes
func (f *FilesystemStorage) Cleanup() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	entries, err := os.ReadDir(f.basePath)
	if err != nil {
		return fmt.Errorf("failed to read base directory: %w", err)
	}

	now := time.Now()
	var cleaned int

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pasteDir := filepath.Join(f.basePath, entry.Name())
		metadataPath := filepath.Join(pasteDir, "metadata.json")

		// Read metadata to check expiration
		if metadataBytes, err := os.ReadFile(metadataPath); err == nil {
			var paste Paste
			if err := json.Unmarshal(metadataBytes, &paste); err == nil {
				if paste.ExpiresAt != nil && now.After(*paste.ExpiresAt) {
					if err := os.RemoveAll(pasteDir); err == nil {
						cleaned++
					}
				}
			}
		}
	}

	fmt.Printf("Cleaned up %d expired pastes\n", cleaned)
	return nil
}

// Close closes the storage backend
func (f *FilesystemStorage) Close() error {
	// Filesystem storage doesn't need explicit closing
	return nil
}

// getCreationTime tries to determine the creation time of a paste directory
func (f *FilesystemStorage) getCreationTime(pasteDir string) time.Time {
	if info, err := os.Stat(pasteDir); err == nil {
		return info.ModTime()
	}
	return time.Now()
}

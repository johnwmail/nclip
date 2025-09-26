package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"log"

	"github.com/johnwmail/nclip/models"
)

type FilesystemStore struct {
	dataDir string
	mu      sync.Mutex
}

func NewFilesystemStore() (*FilesystemStore, error) {
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	return &FilesystemStore{dataDir: dataDir}, nil
}

func (fs *FilesystemStore) Store(paste *models.Paste) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	// Write content
	contentPath := filepath.Join(fs.dataDir, paste.ID)
	if err := os.WriteFile(contentPath, []byte{}, 0o644); err != nil {
		log.Printf("[ERROR] FS Store: failed to write empty content for %s: %v", paste.ID, err)
		return err
	}
	// Write metadata
	metaPath := filepath.Join(fs.dataDir, paste.ID+".json")
	metaData, err := json.MarshalIndent(paste, "", "  ")
	if err != nil {
		return err
	}
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
		log.Printf("[ERROR] FS Get: failed to read metadata for %s: %v", id, err)
		return nil, err
	}
	var paste models.Paste
	if err := json.Unmarshal(metaData, &paste); err != nil {
		log.Printf("[ERROR] FS Get: failed to unmarshal metadata for %s: %v", id, err)
		return nil, err
	}
	return &paste, nil
}

func (fs *FilesystemStore) Delete(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	contentPath := filepath.Join(fs.dataDir, id)
	metaPath := filepath.Join(fs.dataDir, id+".json")
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
		return err
	}
	return nil
}

func (fs *FilesystemStore) StoreContent(id string, content []byte) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	contentPath := filepath.Join(fs.dataDir, id)
	if err := os.WriteFile(contentPath, content, 0o644); err != nil {
		log.Printf("[ERROR] FS StoreContent: failed to write content for %s: %v", id, err)
		return err
	}
	return nil
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

package storage

import (
	"errors"
	"os"
	"testing"

	"github.com/johnwmail/nclip/models"
)

// TestSlugCollision verifies that storing a paste with an existing slug does not overwrite the original paste
func TestSlugCollision(t *testing.T) {
	store := NewMockPasteStore()
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("failed to close store: %v", err)
		}
	}()

	slug := "COLLISION"
	paste1 := &models.Paste{ID: slug, Content: []byte("first")}
	paste2 := &models.Paste{ID: slug, Content: []byte("second")}

	// Store first paste
	err := store.Store(paste1)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	// Try to store second paste with same slug
	err = store.Store(paste2)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	// Get paste
	retrieved, err := store.Get(slug)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	// Should still be the first paste (no overwrite)
	if string(retrieved.Content) != "first" {
		t.Errorf("Slug collision: expected 'first', got '%s'", string(retrieved.Content))
	}
}

// MockPasteStore is a mock implementation of PasteStore for testing
type MockPasteStore struct {
	pastes  map[string]*models.Paste
	closed  bool
	content map[string][]byte
}

// NewMockPasteStore creates a new mock paste store
func NewMockPasteStore() *MockPasteStore {
	return &MockPasteStore{
		pastes:  make(map[string]*models.Paste),
		closed:  false,
		content: make(map[string][]byte),
	}
}

// StoreContent saves the raw content for a paste
func (m *MockPasteStore) StoreContent(id string, content []byte) error {
	if m.closed {
		return errors.New("store is closed")
	}
	m.content[id] = content
	return nil
}

// GetContent retrieves the raw content for a paste
func (m *MockPasteStore) GetContent(id string) ([]byte, error) {
	if m.closed {
		return nil, errors.New("store is closed")
	}
	c, ok := m.content[id]
	if !ok {
		return nil, nil
	}
	return c, nil
}

// GetContentPrefix returns up to n bytes of content for testing purposes
func (m *MockPasteStore) GetContentPrefix(id string, n int64) ([]byte, error) {
	if m.closed {
		return nil, errors.New("store is closed")
	}
	if c, ok := m.content[id]; ok {
		if int64(len(c)) <= n {
			return c, nil
		}
		return c[:n], nil
	}
	return nil, nil
}

// StatContent reports whether content exists and its size in the mock store.
func (m *MockPasteStore) StatContent(id string) (bool, int64, error) {
	if m.closed {
		return false, 0, errors.New("store is closed")
	}
	if c, ok := m.content[id]; ok {
		return true, int64(len(c)), nil
	}
	return false, 0, nil
}

func (m *MockPasteStore) Store(paste *models.Paste) error {
	if m.closed {
		return errors.New("store is closed")
	}
	if paste == nil {
		return errors.New("paste cannot be nil")
	}
	if paste.ID == "" {
		return errors.New("paste ID cannot be empty")
	}
	if _, exists := m.pastes[paste.ID]; exists {
		// Do not overwrite existing paste
		return nil
	}
	m.pastes[paste.ID] = paste
	return nil
}

func (m *MockPasteStore) Get(id string) (*models.Paste, error) {
	if m.closed {
		return nil, errors.New("store is closed")
	}
	if id == "" {
		return nil, errors.New("ID cannot be empty")
	}

	paste, exists := m.pastes[id]
	if !exists {
		return nil, nil // Not found, return nil without error (as per interface contract)
	}

	// Return a copy to prevent external modifications
	copyPaste := *paste
	return &copyPaste, nil
}

func (m *MockPasteStore) Exists(id string) (bool, error) {
	if m.closed {
		return false, errors.New("store is closed")
	}
	if id == "" {
		return false, errors.New("ID cannot be empty")
	}
	_, exists := m.pastes[id]
	return exists, nil
}

func (m *MockPasteStore) IncrementReadCount(id string) error {
	if m.closed {
		return errors.New("store is closed")
	}
	if id == "" {
		return errors.New("ID cannot be empty")
	}

	paste, exists := m.pastes[id]
	if !exists {
		return errors.New("paste not found")
	}

	paste.ReadCount++
	return nil
}

func (m *MockPasteStore) Delete(id string) error {
	if m.closed {
		return errors.New("store is closed")
	}
	if id == "" {
		return errors.New("ID cannot be empty")
	}

	_, exists := m.pastes[id]
	if !exists {
		return errors.New("paste not found")
	}

	delete(m.pastes, id)
	return nil
}

func (m *MockPasteStore) Close() error {
	m.closed = true
	return nil
}

// Test helper methods
func (m *MockPasteStore) IsClosed() bool {
	return m.closed
}

func (m *MockPasteStore) Count() int {
	return len(m.pastes)
}

func (m *MockPasteStore) Clear() {
	m.pastes = make(map[string]*models.Paste)
}

// TestMockPasteStoreImplementation verifies MockPasteStore implements PasteStore interface
func TestMockPasteStoreImplementation(t *testing.T) {
	// Compile-time check that MockPasteStore implements PasteStore
	var _ PasteStore = (*MockPasteStore)(nil)
	t.Log("MockPasteStore correctly implements PasteStore interface")
}

// TestMockPasteStore tests the mock implementation
func TestMockPasteStore(t *testing.T) {
	store := NewMockPasteStore()
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("failed to close store: %v", err)
		}
	}()

	t.Run("Store and Get", func(t *testing.T) {
		paste := &models.Paste{
			ID:      "TEST_STORE_GET",
			Content: []byte("test content"),
		}

		// Store paste
		err := store.Store(paste)
		if err != nil {
			t.Errorf("Store failed: %v", err)
		}

		// Get paste
		retrieved, err := store.Get("TEST_STORE_GET")
		if err != nil {
			t.Errorf("Get failed: %v", err)
		}
		if retrieved == nil {
			t.Error("Get returned nil for existing paste")
		}
		if retrieved != nil && retrieved.ID != paste.ID {
			t.Errorf("Retrieved paste ID mismatch: expected %s, got %s", paste.ID, retrieved.ID)
		}
	})

	t.Run("Closed Store", func(t *testing.T) {
		closedStore := NewMockPasteStore()
		if err := closedStore.Close(); err != nil {
			t.Fatalf("failed to close closedStore: %v", err)
		}

		paste := &models.Paste{ID: "TEST", Content: []byte("test")}

		err := closedStore.Store(paste)
		if err == nil {
			t.Error("Store should return error for closed store")
		}

		_, err = closedStore.Get("TEST")
		if err == nil {
			t.Error("Get should return error for closed store")
		}

		err = closedStore.IncrementReadCount("TEST")
		if err == nil {
			t.Error("IncrementReadCount should return error for closed store")
		}

		err = closedStore.Delete("TEST")
		if err == nil {
			t.Error("Delete should return error for closed store")
		}
	})
}

// Integration tests for FilesystemStore (local FS mode)
// helper to setup a local FilesystemStore for tests
func setupLocalFilesystem(t *testing.T) (PasteStore, func()) {
	if err := os.Setenv("NCLIP_S3_BUCKET", ""); err != nil { // Ensure S3 is disabled
		t.Fatalf("failed to set NCLIP_S3_BUCKET: %v", err)
	}

	if err := os.MkdirAll("./testdata", 0o755); err != nil {
		t.Fatalf("failed to create testdata: %v", err)
	}

	store, err := NewFilesystemStore("./testdata")
	if err != nil {
		t.Fatalf("Failed to create FilesystemStore: %v", err)
	}

	cleanup := func() {
		if err := store.Close(); err != nil {
			t.Fatalf("failed to close store: %v", err)
		}
		if err := os.RemoveAll("./testdata"); err != nil {
			t.Fatalf("failed to remove testdata: %v", err)
		}
	}

	return store, cleanup
}

func TestFilesystemStore_LocalFS_StoreAndGet(t *testing.T) {
	store, cleanup := setupLocalFilesystem(t)
	defer cleanup()

	paste := &models.Paste{
		ID:          "FS_TEST",
		Content:     []byte("hello fs"),
		ContentType: "text/plain",
		Size:        int64(len([]byte("hello fs"))),
	}

	// Store
	if err := store.Store(paste); err != nil {
		t.Fatalf("FilesystemStore.Store failed: %v", err)
	}

	// Get
	retrieved, err := store.Get("FS_TEST")
	if err != nil {
		t.Fatalf("FilesystemStore.Get failed: %v", err)
	}
	if retrieved == nil || retrieved.ID != paste.ID {
		t.Fatalf("FilesystemStore.Get returned wrong paste: %+v", retrieved)
	}
}

func TestFilesystemStore_LocalFS_ReadContentAndCleanup(t *testing.T) {
	store, cleanup := setupLocalFilesystem(t)
	defer cleanup()

	paste := &models.Paste{ID: "FS_TEST2", Size: 1}
	if err := store.Store(paste); err != nil {
		t.Fatalf("FilesystemStore.Store failed: %v", err)
	}

	// IncrementReadCount
	if err := store.IncrementReadCount("FS_TEST2"); err != nil {
		t.Fatalf("FilesystemStore.IncrementReadCount failed: %v", err)
	}
	retrieved, _ := store.Get("FS_TEST2")
	if retrieved.ReadCount != 1 {
		t.Fatalf("FilesystemStore.ReadCount should be 1, got %d", retrieved.ReadCount)
	}

	// StoreContent + GetContent
	if err := store.StoreContent("FS_TEST2", []byte("raw content")); err != nil {
		t.Fatalf("FilesystemStore.StoreContent failed: %v", err)
	}
	content, err := store.GetContent("FS_TEST2")
	if err != nil {
		t.Fatalf("FilesystemStore.GetContent failed: %v", err)
	}
	if string(content) != "raw content" {
		t.Fatalf("FilesystemStore.GetContent returned wrong content: %s", string(content))
	}
}

func TestFilesystemStore_LocalFS_Delete(t *testing.T) {
	store, cleanup := setupLocalFilesystem(t)
	defer cleanup()

	paste := &models.Paste{ID: "FS_TEST3", Size: 1}
	if err := store.Store(paste); err != nil {
		t.Fatalf("FilesystemStore.Store failed: %v", err)
	}

	// Delete
	if err := store.Delete("FS_TEST3"); err != nil {
		t.Fatalf("FilesystemStore.Delete failed: %v", err)
	}
	retrieved, err := store.Get("FS_TEST3")
	if err == nil && retrieved != nil {
		t.Fatalf("FilesystemStore.Get should return nil after delete, got %+v", retrieved)
	}
}

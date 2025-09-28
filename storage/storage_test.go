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
	defer store.Close()

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
	defer store.Close()

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
		closedStore.Close()

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
func TestFilesystemStore_LocalFS(t *testing.T) {
	os.Setenv("NCLIP_S3_BUCKET", "") // Ensure S3 is disabled
	os.Setenv("NCLIP_DATA_DIR", "./testdata")

	_ = os.Mkdir("./testdata", 0o755)
	defer os.RemoveAll("./testdata")

	store, err := NewFilesystemStore()
	if err != nil {
		t.Fatalf("Failed to create FilesystemStore: %v", err)
	}
	defer store.Close()

	paste := &models.Paste{
		ID:          "FS_TEST",
		Content:     []byte("hello fs"),
		ContentType: "text/plain",
		Size:        int64(len([]byte("hello fs"))),
	}

	// Store
	err = store.Store(paste)
	if err != nil {
		t.Errorf("FilesystemStore.Store failed: %v", err)
	}

	// Get
	retrieved, err := store.Get("FS_TEST")
	if err != nil {
		t.Errorf("FilesystemStore.Get failed: %v", err)
	}
	if retrieved == nil || retrieved.ID != paste.ID {
		t.Errorf("FilesystemStore.Get returned wrong paste: %+v", retrieved)
	}

	// IncrementReadCount
	err = store.IncrementReadCount("FS_TEST")
	if err != nil {
		t.Errorf("FilesystemStore.IncrementReadCount failed: %v", err)
	}
	retrieved, _ = store.Get("FS_TEST")
	if retrieved.ReadCount != 1 {
		t.Errorf("FilesystemStore.ReadCount should be 1, got %d", retrieved.ReadCount)
	}

	// StoreContent
	err = store.StoreContent("FS_TEST", []byte("raw content"))
	if err != nil {
		t.Errorf("FilesystemStore.StoreContent failed: %v", err)
	}

	// GetContent
	content, err := store.GetContent("FS_TEST")
	if err != nil {
		t.Errorf("FilesystemStore.GetContent failed: %v", err)
	}
	if string(content) != "raw content" {
		t.Errorf("FilesystemStore.GetContent returned wrong content: %s", string(content))
	}

	// Delete
	err = store.Delete("FS_TEST")
	if err != nil {
		t.Errorf("FilesystemStore.Delete failed: %v", err)
	}
	retrieved, err = store.Get("FS_TEST")
	if err == nil && retrieved != nil {
		t.Errorf("FilesystemStore.Get should return nil after delete, got %+v", retrieved)
	}

}

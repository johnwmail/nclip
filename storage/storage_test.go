package storage

import (
	"errors"
	"testing"

	"github.com/johnwmail/nclip/models"
)

// MockPasteStore is a mock implementation of PasteStore for testing
type MockPasteStore struct {
	pastes map[string]*models.Paste
	closed bool
}

// NewMockPasteStore creates a new mock paste store
func NewMockPasteStore() *MockPasteStore {
	return &MockPasteStore{
		pastes: make(map[string]*models.Paste),
		closed: false,
	}
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

	t.Run("IncrementReadCount", func(t *testing.T) {
		paste := &models.Paste{
			ID:      "TEST_READ_COUNT",
			Content: []byte("test content"),
		}

		store.Store(paste)

		// Increment read count
		err := store.IncrementReadCount("TEST_READ_COUNT")
		if err != nil {
			t.Errorf("IncrementReadCount failed: %v", err)
		}

		// Verify read count
		retrieved, _ := store.Get("TEST_READ_COUNT")
		if retrieved.ReadCount != 1 {
			t.Errorf("Read count should be 1, got %d", retrieved.ReadCount)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		paste := &models.Paste{
			ID:      "TEST_DELETE",
			Content: []byte("test content"),
		}

		store.Store(paste)

		// Delete paste
		err := store.Delete("TEST_DELETE")
		if err != nil {
			t.Errorf("Delete failed: %v", err)
		}

		// Verify deletion
		retrieved, err := store.Get("TEST_DELETE")
		if err != nil {
			t.Errorf("Get after delete should not error: %v", err)
		}
		if retrieved != nil {
			t.Error("Get should return nil for deleted paste")
		}
	})

	t.Run("Error Cases", func(t *testing.T) {
		// Test nil paste
		err := store.Store(nil)
		if err == nil {
			t.Error("Store should return error for nil paste")
		}

		// Test empty ID
		err = store.Store(&models.Paste{ID: "", Content: []byte("test")})
		if err == nil {
			t.Error("Store should return error for empty ID")
		}

		// Test get with empty ID
		_, err = store.Get("")
		if err == nil {
			t.Error("Get should return error for empty ID")
		}

		// Test increment read count for non-existing paste
		err = store.IncrementReadCount("NON_EXISTING")
		if err == nil {
			t.Error("IncrementReadCount should return error for non-existing paste")
		}

		// Test delete non-existing paste
		err = store.Delete("NON_EXISTING")
		if err == nil {
			t.Error("Delete should return error for non-existing paste")
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

	t.Run("Helper Methods", func(t *testing.T) {
		testStore := NewMockPasteStore()

		// Test Count
		if testStore.Count() != 0 {
			t.Errorf("New store should have count 0, got %d", testStore.Count())
		}

		// Add a paste
		paste := &models.Paste{ID: "COUNT_TEST", Content: []byte("test")}
		testStore.Store(paste)

		if testStore.Count() != 1 {
			t.Errorf("Store with one paste should have count 1, got %d", testStore.Count())
		}

		// Test Clear
		testStore.Clear()
		if testStore.Count() != 0 {
			t.Errorf("Store after clear should have count 0, got %d", testStore.Count())
		}

		// Test IsClosed
		if testStore.IsClosed() {
			t.Error("New store should not be closed")
		}

		testStore.Close()
		if !testStore.IsClosed() {
			t.Error("Store should be closed after Close()")
		}
	})

}

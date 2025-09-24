package storage

import (
	"testing"

	"github.com/johnwmail/nclip/models"
)

// TestPasteStoreInterface verifies that the PasteStore interface is well-defined
func TestPasteStoreInterface(t *testing.T) {
	// This is a compile-time test to ensure the interface is properly defined
	// Any type implementing PasteStore must have all these methods

	// Test that MockPasteStore implements the interface
	var _ PasteStore = (*MockPasteStore)(nil)

	t.Log("PasteStore interface is properly defined")
}

// TestPasteStoreInterfaceUsage tests that the interface can be used polymorphically
func TestPasteStoreInterfaceUsage(t *testing.T) {
	// Create a mock store
	mockStore := NewMockPasteStore()

	// Use it through the interface
	var store PasteStore = mockStore

	// Test basic operations through interface
	paste := &models.Paste{
		ID:      "INTERFACE_TEST",
		Content: []byte("test content"),
	}

	// Store
	err := store.Store(paste)
	if err != nil {
		t.Errorf("Interface Store failed: %v", err)
	}

	// Get
	retrieved, err := store.Get("INTERFACE_TEST")
	if err != nil {
		t.Errorf("Interface Get failed: %v", err)
	}
	if retrieved == nil {
		t.Error("Interface Get returned nil")
	}
	if retrieved != nil && retrieved.ID != paste.ID {
		t.Errorf("Retrieved paste ID mismatch: expected %s, got %s", paste.ID, retrieved.ID)
	}

	// IncrementReadCount
	err = store.IncrementReadCount("INTERFACE_TEST")
	if err != nil {
		t.Errorf("Interface IncrementReadCount failed: %v", err)
	}

	// Delete
	err = store.Delete("INTERFACE_TEST")
	if err != nil {
		t.Errorf("Interface Delete failed: %v", err)
	}

	// Close
	err = store.Close()
	if err != nil {
		t.Errorf("Interface Close failed: %v", err)
	}
}

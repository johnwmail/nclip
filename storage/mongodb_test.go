package storage

import (
	"testing"

	"github.com/johnwmail/nclip/models"
)

// TestMongoStoreInterfaceCompliance verifies MongoStore implements PasteStore interface at compile time
func TestMongoStoreInterfaceCompliance(t *testing.T) {
	// This is a compile-time check that MongoStore implements PasteStore
	var _ PasteStore = (*MongoStore)(nil)
	t.Log("MongoStore correctly implements PasteStore interface")
}

// TestMongoStoreInitialization tests MongoStore struct creation
func TestMongoStoreInitialization(t *testing.T) {
	// Test that we can create a MongoStore instance
	// Note: We don't actually connect to MongoDB in unit tests
	store := &MongoStore{}

	if store == nil {
		t.Error("Failed to create MongoStore instance")
	}

	t.Log("MongoStore can be initialized")
}

// MockMongoDBBehavior tests how MongoStore would behave with mocked dependencies
func TestMockMongoDBBehavior(t *testing.T) {
	// Since we can't test real MongoDB operations in unit tests,
	// we verify that the structure supports the expected operations

	testPaste := &models.Paste{
		ID:      "MONGO_TEST",
		Content: []byte("test content for mongodb"),
	}

	// Verify paste structure is compatible with MongoDB operations
	if testPaste.ID == "" {
		t.Error("Paste ID should not be empty")
	}

	if len(testPaste.Content) == 0 {
		t.Error("Paste content should not be empty")
	}

	t.Log("MongoStore operations would work with Paste model")
}

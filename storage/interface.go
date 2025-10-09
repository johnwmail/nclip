package storage

import "github.com/johnwmail/nclip/models"

// PasteStore defines the interface for paste storage backends
type PasteStore interface {
	// Store saves a paste to the storage backend
	Store(paste *models.Paste) error

	// Get retrieves a paste by its ID
	Get(id string) (*models.Paste, error)

	// Exists checks if a paste exists by its ID
	Exists(id string) (bool, error)

	// Delete removes a paste from storage
	Delete(id string) error

	// IncrementReadCount increments the read count for a paste
	IncrementReadCount(id string) error

	// Close closes the storage connection
	Close() error

	// StoreContent saves the raw content for a paste
	StoreContent(id string, content []byte) error

	// GetContent retrieves the raw content for a paste
	GetContent(id string) ([]byte, error)

	// GetContentPrefix retrieves up to n bytes of the raw content for a paste.
	// Implementations should return as many bytes as are available up to n.
	GetContentPrefix(id string, n int64) ([]byte, error)
}

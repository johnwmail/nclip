package storage

import (
	"time"
)

// Paste represents a stored paste
type Paste struct {
	ID          string            `json:"id"`
	Content     []byte            `json:"content"`
	ContentType string            `json:"content_type"`
	Filename    string            `json:"filename,omitempty"`
	Language    string            `json:"language,omitempty"`
	Title       string            `json:"title,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
	ClientIP    string            `json:"client_ip"`
	Size        int64             `json:"size"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Storage defines the interface for paste storage backends
type Storage interface {
	// Store saves a paste and returns its ID
	Store(paste *Paste) error

	// Get retrieves a paste by ID
	Get(id string) (*Paste, error)

	// Exists checks if a paste with the given ID exists
	Exists(id string) bool

	// Delete removes a paste by ID
	Delete(id string) error

	// List returns a list of paste IDs, optionally limited
	List(limit int) ([]string, error)

	// Stats returns storage statistics
	Stats() (*Stats, error)

	// Cleanup removes expired pastes
	Cleanup() error

	// Close closes the storage backend
	Close() error
}

// Stats represents storage statistics
type Stats struct {
	TotalPastes   int64 `json:"total_pastes"`
	TotalSize     int64 `json:"total_size_bytes"`
	ExpiredPastes int64 `json:"expired_pastes"`
}

// PasteRequest represents a request to create a new paste
type PasteRequest struct {
	Content     []byte
	ContentType string
	Filename    string
	Language    string
	Title       string
	ExpiresIn   time.Duration
	ClientIP    string
	Metadata    map[string]string
}

package models
package models

import (
	"time"
)

// Paste represents a paste/clipboard entry in the system
type Paste struct {
	ID           string     `json:"id" bson:"_id"`
	CreatedAt    time.Time  `json:"created_at" bson:"created_at"`
	ExpiresAt    *time.Time `json:"expires_at" bson:"expires_at,omitempty"`
	Size         int64      `json:"size" bson:"size"`
	ContentType  string     `json:"content_type" bson:"content_type"`
	BurnAfterRead bool      `json:"burn_after_read" bson:"burn_after_read"`
	ReadCount    int        `json:"read_count" bson:"read_count"`
	Content      []byte     `json:"-" bson:"content"` // Not exposed in JSON
}

// IsExpired checks if the paste has expired
func (p *Paste) IsExpired() bool {
	if p.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*p.ExpiresAt)
}

// ShouldBurn returns true if this paste should be deleted after reading
func (p *Paste) ShouldBurn() bool {
	return p.BurnAfterRead && p.ReadCount > 0
}
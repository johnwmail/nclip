package services

import (
	"fmt"
	"time"

	"github.com/johnwmail/nclip/config"
	"github.com/johnwmail/nclip/models"
	"github.com/johnwmail/nclip/storage"
	"github.com/johnwmail/nclip/utils"
)

// PasteService handles paste business logic
type PasteService struct {
	store  storage.PasteStore
	config *config.Config
}

// NewPasteService creates a new paste service
func NewPasteService(store storage.PasteStore, config *config.Config) *PasteService {
	return &PasteService{
		store:  store,
		config: config,
	}
}

// CreatePasteRequest represents a request to create a paste
type CreatePasteRequest struct {
	Content       []byte
	Filename      string
	CustomSlug    string
	BurnAfterRead bool
	TTL           time.Duration
}

// CreatePasteResponse represents the response from creating a paste
type CreatePasteResponse struct {
	Slug string
	URL  string
}

// GenerateSlug generates a unique slug for a paste
func (s *PasteService) GenerateSlug() (string, error) {
	batchSize := 5
	lengths := []int{5, 6, 7}
	var slug string
	for _, length := range lengths {
		candidates, err := utils.GenerateSlugBatch(batchSize, length)
		if err != nil {
			return "", fmt.Errorf("failed to generate slug batch: %w", err)
		}
		for _, candidate := range candidates {
			exists, err := s.store.Exists(candidate)
			if err != nil {
				continue // skip on error
			}
			if !exists {
				slug = candidate
				return slug, nil
			}
			// exists, check if expired
			existing, err := s.store.Get(candidate)
			if err != nil || existing == nil || existing.IsExpired() {
				slug = candidate
				return slug, nil
			}
		}
	}
	return "", fmt.Errorf("failed to generate unique slug after 3 batches")
}

// ValidateCustomSlug validates and checks if a custom slug is available
func (s *PasteService) ValidateCustomSlug(slug string) error {
	if !utils.IsValidSlug(slug) {
		return fmt.Errorf("invalid slug format")
	}

	exists, err := s.store.Exists(slug)
	if err != nil {
		return fmt.Errorf("failed to check slug existence: %w", err)
	}
	if exists {
		existing, err := s.store.Get(slug)
		if err != nil {
			return fmt.Errorf("failed to retrieve existing paste: %w", err)
		}
		if existing != nil && !existing.IsExpired() {
			return fmt.Errorf("slug already exists")
		}
	}
	return nil
}

// CreatePaste creates a new paste
func (s *PasteService) CreatePaste(req CreatePasteRequest) (*CreatePasteResponse, error) {
	var slug string
	var err error

	if req.CustomSlug != "" {
		if err := s.ValidateCustomSlug(req.CustomSlug); err != nil {
			return nil, err
		}
		slug = req.CustomSlug
	} else {
		slug, err = s.GenerateSlug()
		if err != nil {
			return nil, fmt.Errorf("failed to generate slug: %w", err)
		}
	}

	expiresAt := time.Now().Add(req.TTL)

	contentType := utils.DetectContentType(req.Filename, req.Content)
	paste := &models.Paste{
		ID:            slug,
		CreatedAt:     time.Now(),
		ExpiresAt:     &expiresAt,
		Size:          int64(len(req.Content)),
		ContentType:   contentType,
		BurnAfterRead: req.BurnAfterRead,
		ReadCount:     0,
	}

	if err := s.store.StoreContent(slug, req.Content); err != nil {
		return nil, fmt.Errorf("failed to store content: %w", err)
	}
	if err := s.store.Store(paste); err != nil {
		return nil, fmt.Errorf("failed to store metadata: %w", err)
	}

	return &CreatePasteResponse{
		Slug: slug,
		URL:  "", // Will be set by handler based on request context
	}, nil
}

// GetPaste retrieves a paste by slug
func (s *PasteService) GetPaste(slug string) (*models.Paste, error) {
	paste, err := s.store.Get(slug)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve paste: %w", err)
	}
	if paste == nil {
		return nil, fmt.Errorf("paste not found")
	}
	if paste.IsExpired() {
		return nil, fmt.Errorf("paste expired")
	}

	return paste, nil
}

// GetPasteContent retrieves paste content
func (s *PasteService) GetPasteContent(slug string) ([]byte, error) {
	content, err := s.store.GetContent(slug)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve content: %w", err)
	}
	return content, nil
}

// IncrementReadCount increments the read count for a paste
func (s *PasteService) IncrementReadCount(slug string) error {
	return s.store.IncrementReadCount(slug)
}

// DeletePaste deletes a paste
func (s *PasteService) DeletePaste(slug string) error {
	return s.store.Delete(slug)
}

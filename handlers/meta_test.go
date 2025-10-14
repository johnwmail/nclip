package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/models"
)

// MockPasteStore implements storage.PasteStore for testing
type MockPasteStore struct {
	pastes  map[string]*models.Paste
	getErr  error
	content map[string][]byte
}

func NewMockPasteStore() *MockPasteStore {
	return &MockPasteStore{
		pastes:  make(map[string]*models.Paste),
		content: make(map[string][]byte),
	}
}

// StoreContent saves the raw content for a paste
func (m *MockPasteStore) StoreContent(id string, content []byte) error {
	m.content[id] = content
	return nil
}

// GetContent retrieves the raw content for a paste
func (m *MockPasteStore) GetContent(id string) ([]byte, error) {
	c, ok := m.content[id]
	if !ok {
		return nil, nil
	}
	return c, nil
}

func (m *MockPasteStore) Store(paste *models.Paste) error {
	m.pastes[paste.ID] = paste
	return nil
}

func (m *MockPasteStore) Get(id string) (*models.Paste, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.pastes[id], nil
}

func (m *MockPasteStore) Exists(id string) (bool, error) {
	_, exists := m.pastes[id]
	return exists, nil
}

func (m *MockPasteStore) Delete(id string) error {
	delete(m.pastes, id)
	return nil
}

func (m *MockPasteStore) IncrementReadCount(id string) error {
	if paste := m.pastes[id]; paste != nil {
		paste.ReadCount++
	}
	return nil
}

func (m *MockPasteStore) Close() error {
	return nil
}

func (m *MockPasteStore) GetContentPrefix(id string, n int64) ([]byte, error) {
	c, ok := m.content[id]
	if !ok {
		return nil, nil
	}
	if int64(len(c)) <= n {
		return c, nil
	}
	return c[:n], nil
}

func (m *MockPasteStore) StatContent(id string) (bool, int64, error) {
	if c, ok := m.content[id]; ok {
		return true, int64(len(c)), nil
	}
	return false, 0, nil
}

func (m *MockPasteStore) SetGetError(err error) {
	m.getErr = err
}

func TestMetaHandler_GetMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		slug           string
		setupStore     func(*MockPasteStore)
		expectedStatus int
		expectedBody   map[string]interface{}
		expectError    bool
	}{
		{
			name: "valid paste found",
			slug: "ABC23",
			setupStore: func(store *MockPasteStore) {
				now := time.Now()
				expiresAt := now.Add(1 * time.Hour)
				paste := &models.Paste{
					ID:            "ABC23",
					CreatedAt:     now,
					ExpiresAt:     &expiresAt,
					Size:          100,
					ContentType:   "text/plain",
					BurnAfterRead: false,
					ReadCount:     5,
					Content:       []byte("test content"),
				}
				store.Store(paste)
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "invalid slug format",
			slug:           "invalid-slug!",
			setupStore:     func(store *MockPasteStore) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "Invalid slug format",
			},
			expectError: true,
		},
		{
			name:           "paste not found",
			slug:           "XYZ89",
			setupStore:     func(store *MockPasteStore) {},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"error": "Paste not found",
			},
			expectError: true,
		},
		{
			name: "store error",
			slug: "ERR23",
			setupStore: func(store *MockPasteStore) {
				store.SetGetError(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "Failed to retrieve paste",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock store
			store := NewMockPasteStore()
			tt.setupStore(store)

			// Create handler
			handler := NewMetaHandler(store)

			// Setup request
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{
				{Key: "slug", Value: tt.slug},
			}

			// Execute handler
			handler.GetMetadata(c)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check response body
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if tt.expectError {
				// Check for error message
				if errorMsg, ok := response["error"]; !ok {
					t.Errorf("Expected error field in response")
				} else if errorMsg != tt.expectedBody["error"] {
					t.Errorf("Expected error '%v', got '%v'", tt.expectedBody["error"], errorMsg)
				}
			} else {
				// Check for metadata fields (not error)
				expectedFields := []string{"id", "created_at", "expires_at", "size", "content_type", "burn_after_read", "read_count"}
				for _, field := range expectedFields {
					if _, ok := response[field]; !ok {
						t.Errorf("Expected field '%s' in response", field)
					}
				}

				// Ensure content is not included
				if _, ok := response["content"]; ok {
					t.Errorf("Content should not be included in metadata response")
				}
			}
		})
	}
}

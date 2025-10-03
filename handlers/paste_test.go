package handlers

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/config"
	"github.com/johnwmail/nclip/models"
)

// takenStore implements storage.PasteStore, always returns a taken paste
type takenStore struct{}

func (s *takenStore) Get(id string) (*models.Paste, error) {
	return &models.Paste{ID: id, ExpiresAt: nil}, nil
}
func (s *takenStore) Exists(id string) (bool, error)    { return true, nil }
func (s *takenStore) Store(*models.Paste) error         { return nil }
func (s *takenStore) StoreContent(string, []byte) error { return nil }
func (s *takenStore) GetContent(string) ([]byte, error) { return nil, nil }
func (s *takenStore) Delete(string) error               { return nil }
func (s *takenStore) IncrementReadCount(string) error   { return nil }
func (s *takenStore) Close() error                      { return nil }

func TestSlugCollisionExhaustion(t *testing.T) {
	// Mock GenerateSlugBatch to always return the same 5 slugs
	fixedSlugs := []string{"AAAAA", "BBBBB", "CCCCC", "DDDDD", "EEEEE"}
	mockGen := func(batchSize, length int) ([]string, error) {
		return fixedSlugs, nil
	}

	cfg := &config.Config{
		URL:        "http://localhost:8080",
		SlugLength: 5,
		DefaultTTL: 3600,
		BufferSize: 5 * 1024 * 1024,
		Version:    "test",
		BuildTime:  "test-time",
		CommitHash: "test-hash",
	}
	store := &takenStore{}
	handler := NewPasteHandler(store, cfg)
	handler.GenerateSlugBatch = mockGen

	// Setup Gin router for POST /
	r := gin.New()
	r.POST("/", handler.Upload)

	// Make POST request
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", strings.NewReader("test content"))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 error, got %d", w.Code)
	}
}

// setupTestHandler creates a test handler for testing
func setupTestHandler() *PasteHandler {
	cfg := &config.Config{
		URL:        "https://example.com",
		SlugLength: 5,
		DefaultTTL: 3600,
		BufferSize: 5 * 1024 * 1024,
		Version:    "test",
		BuildTime:  "test-time",
		CommitHash: "test-hash",
	}
	return NewPasteHandler(nil, cfg) // store is nil for utility function tests
}

func TestPasteHandler_isCli(t *testing.T) {
	handler := setupTestHandler()

	tests := []struct {
		name      string
		userAgent string
		want      bool
	}{
		{
			name:      "curl user agent",
			userAgent: "curl/7.68.0",
			want:      true,
		},
		{
			name:      "wget user agent",
			userAgent: "Wget/1.20.3 (linux-gnu)",
			want:      true,
		},
		{
			name:      "powershell user agent",
			userAgent: "Mozilla/5.0 (Windows NT; Windows NT 10.0; en-US) WindowsPowerShell/5.1.17763.1007",
			want:      true,
		},
		{
			name:      "chrome browser",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			want:      false,
		},
		{
			name:      "firefox browser",
			userAgent: "Mozilla/5.0 (X11; Linux x86_64; rv:91.0) Gecko/20100101 Firefox/91.0",
			want:      false,
		},
		{
			name:      "empty user agent",
			userAgent: "",
			want:      false,
		},
		{
			name:      "case insensitive curl",
			userAgent: "CURL/7.68.0",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request with the user agent
			req, _ := http.NewRequest("GET", "/", nil)
			req.Header.Set("User-Agent", tt.userAgent)

			// Create test context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			got := handler.isCli(c)
			if got != tt.want {
				t.Errorf("PasteHandler.isCli() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPasteHandler_isHTTPS(t *testing.T) {
	handler := setupTestHandler()

	tests := []struct {
		name    string
		headers map[string]string
		hasTLS  bool
		want    bool
	}{
		{
			name:    "direct TLS connection",
			headers: map[string]string{},
			hasTLS:  true,
			want:    true,
		},
		{
			name: "X-Forwarded-Proto https",
			headers: map[string]string{
				"X-Forwarded-Proto": "https",
			},
			hasTLS: false,
			want:   true,
		},
		{
			name:    "no indicators - plain HTTP",
			headers: map[string]string{},
			hasTLS:  false,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request
			req, _ := http.NewRequest("GET", "/", nil)

			// Set headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Create test context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Mock TLS if needed
			if tt.hasTLS {
				c.Request.TLS = &tls.ConnectionState{}
			}

			got := handler.isHTTPS(c)
			if got != tt.want {
				t.Errorf("PasteHandler.isHTTPS() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBufferSizeLimit(t *testing.T) {
	cfg := &config.Config{
		URL:        "http://localhost:8080",
		SlugLength: 5,
		DefaultTTL: 3600,
		BufferSize: 1024, // Set small buffer size for testing
		Version:    "test",
		BuildTime:  "test-time",
		CommitHash: "test-hash",
	}
	store := &takenStore{}
	handler := NewPasteHandler(store, cfg)

	// Test content larger than buffer size
	largeContent := make([]byte, 2048) // 2KB, larger than 1KB limit
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	// Test direct POST upload
	t.Run("direct_post_large_content", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/", bytes.NewReader(largeContent))
		c.Request.Header.Set("Content-Type", "text/plain")

		handler.Upload(c)

		if w.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("expected 413 Payload Too Large for large content, got %d", w.Code)
		}

		var response map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("failed to parse response JSON: %v", err)
		}

		if response["error"] == "" {
			t.Errorf("expected error message in response")
		}

		if !strings.Contains(response["error"], "content too large") {
			t.Errorf("expected 'content too large' error, got: %s", response["error"])
		}
	})

	// Test multipart upload with large file
	t.Run("multipart_large_file", func(t *testing.T) {
		// Create multipart form data
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("file", "large.txt")
		if err != nil {
			t.Fatal(err)
		}
		part.Write(largeContent)
		writer.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/", body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())

		handler.Upload(c)

		if w.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("expected 413 Payload Too Large for large file, got %d", w.Code)
		}

		var response map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("failed to parse response JSON: %v", err)
		}

		if response["error"] == "" {
			t.Errorf("expected error message in response")
		}

		if !strings.Contains(response["error"], "content too large") {
			t.Errorf("expected 'content too large' error, got: %s", response["error"])
		}
	})
}

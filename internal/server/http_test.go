package server

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/johnwmail/nclip/internal/config"
	"github.com/johnwmail/nclip/internal/storage"
)

// Helper function to create a test logger that suppresses output
func createTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNewHTTPServer(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.OutputDir = t.TempDir()

	store, err := storage.NewFilesystemStorage(cfg.OutputDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Logf("Failed to close storage: %v", err)
		}
	}()

	server := NewHTTPServer(cfg, store, createTestLogger())
	if server == nil {
		t.Fatal("Expected non-nil server")
	}
}

func TestHTTPServer_HandlePost(t *testing.T) {
	// Setup
	cfg := config.DefaultConfig()
	cfg.OutputDir = t.TempDir()
	cfg.BaseURL = "http://test.example.com:8080/"
	cfg.HTTPPort = 8080

	store, err := storage.NewFilesystemStorage(cfg.OutputDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Logf("Failed to close storage: %v", err)
		}
	}()

	server := NewHTTPServer(cfg, store, createTestLogger())

	testCases := []struct {
		name           string
		method         string
		body           string
		contentType    string
		filename       string
		expectedStatus int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:           "valid plain text paste",
			method:         "POST",
			body:           "Hello, World!\nThis is a test paste.",
			contentType:    "text/plain",
			filename:       "test.txt",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				if !strings.Contains(body, "http://test.example.com:8080/") {
					t.Errorf("Response should contain full URL, got: %s", body)
				}
			},
		},
		{
			name:           "empty body",
			method:         "POST",
			body:           "",
			contentType:    "text/plain",
			filename:       "",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				if !strings.Contains(body, "Empty paste") {
					t.Errorf("Expected 'Empty paste' error, got: %s", body)
				}
			},
		},
		{
			name:           "json content",
			method:         "POST",
			body:           `{"key": "value", "number": 42}`,
			contentType:    "application/json",
			filename:       "data.json",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				if !strings.Contains(body, "http://test.example.com:8080/") {
					t.Errorf("Response should contain URL, got: %s", body)
				}
			},
		},
		{
			name:           "code content",
			method:         "POST",
			body:           "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, Go!\")\n}",
			contentType:    "text/plain",
			filename:       "main.go",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				if !strings.Contains(body, "http://") {
					t.Errorf("Response should contain URL, got: %s", body)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			if tc.filename != "" {
				req.Header.Set("X-Filename", tc.filename)
			}

			w := httptest.NewRecorder()
			handler := server.TestHandler()
			handler(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}

			if tc.checkResponse != nil {
				tc.checkResponse(t, w.Body.String())
			}
		})
	}
}

func TestHTTPServer_HandleGet(t *testing.T) {
	// Setup
	cfg := config.DefaultConfig()
	cfg.OutputDir = t.TempDir()

	store, err := storage.NewFilesystemStorage(cfg.OutputDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Logf("Failed to close storage: %v", err)
		}
	}()

	server := NewHTTPServer(cfg, store, createTestLogger())

	// Create a test paste
	paste := &storage.Paste{
		ID:          "test123",
		Content:     []byte("Hello, World!\nThis is a test paste."),
		ContentType: "text/plain",
		Filename:    "test.txt",
		CreatedAt:   time.Now(),
		ClientIP:    "127.0.0.1",
		Size:        int64(len("Hello, World!\nThis is a test paste.")),
	}

	if err := store.Store(paste); err != nil {
		t.Fatalf("Failed to store test paste: %v", err)
	}

	testCases := []struct {
		name           string
		path           string
		expectedStatus int
		checkResponse  func(t *testing.T, body string, contentType string)
	}{
		{
			name:           "get existing paste",
			path:           "/test123",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body, contentType string) {
				if body != "Hello, World!\nThis is a test paste." {
					t.Errorf("Expected paste content, got: %s", body)
				}
				if !strings.Contains(contentType, "text/plain") {
					t.Errorf("Expected text/plain content type, got: %s", contentType)
				}
			},
		},
		{
			name:           "get non-existent paste",
			path:           "/nonexistent",
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body, contentType string) {
				if !strings.Contains(body, "not found") {
					t.Errorf("Expected 'not found' error, got: %s", body)
				}
			},
		},
		{
			name:           "get root path",
			path:           "/",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body, contentType string) {
				if !strings.Contains(contentType, "text/html") {
					t.Errorf("Expected HTML content type for root, got: %s", contentType)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			w := httptest.NewRecorder()

			handler := server.TestHandler()
			handler(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}

			if tc.checkResponse != nil {
				tc.checkResponse(t, w.Body.String(), w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestHTTPServer_HandleOptions(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.OutputDir = t.TempDir()

	store, err := storage.NewFilesystemStorage(cfg.OutputDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer closeStorage(t, store)

	server := NewHTTPServer(cfg, store, createTestLogger())

	req := httptest.NewRequest("OPTIONS", "/", nil)
	w := httptest.NewRecorder()

	handler := server.TestHandler()
	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for OPTIONS (method not allowed), got %d", w.Code)
	}
}

func TestHTTPServer_UnsupportedMethod(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.OutputDir = t.TempDir()

	store, err := storage.NewFilesystemStorage(cfg.OutputDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer closeStorage(t, store)

	server := NewHTTPServer(cfg, store, createTestLogger())

	unsupportedMethods := []string{"PUT", "DELETE", "PATCH", "HEAD"}

	for _, method := range unsupportedMethods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			w := httptest.NewRecorder()

			handler := server.TestHandler()
			handler(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s method, got %d", method, w.Code)
			}
		})
	}
}

func TestHTTPServer_LargePaste(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.OutputDir = t.TempDir()
	cfg.BufferSize = 1024 // Small buffer for testing

	store, err := storage.NewFilesystemStorage(cfg.OutputDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer closeStorage(t, store)

	server := NewHTTPServer(cfg, store, createTestLogger())

	// Create content larger than buffer
	largeContent := strings.Repeat("This is a test line.\n", 100) // ~2KB content

	req := httptest.NewRequest("POST", "/", strings.NewReader(largeContent))
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	handler := server.TestHandler()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for large paste, got %d", w.Code)
	}

	// Verify the paste was stored correctly
	response := w.Body.String()
	if !strings.Contains(response, "http://") {
		t.Errorf("Expected URL in response, got: %s", response)
	}
}

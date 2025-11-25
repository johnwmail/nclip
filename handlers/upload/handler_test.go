package upload

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/config"
	"github.com/johnwmail/nclip/internal/services"
	"github.com/johnwmail/nclip/storage"
)

// base64Test is a lightweight descriptor for the table-driven base64 tests.
type base64Test struct {
	name          string
	content       string
	useBase64     bool
	useRoute      bool // Use /base64 route instead of header
	expectError   bool
	errorContains string
}

// buildTestBody prepares the request body string for a test case.
func buildTestBody(tt base64Test) string {
	if !tt.useBase64 {
		return tt.content
	}
	// When expecting invalid base64, return the raw (invalid) content
	if tt.errorContains == "invalid base64 encoding" {
		return tt.content
	}
	// Empty content encodes to empty string
	if tt.content == "" {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(tt.content))
}

// setupTestRouterForBase64 creates a Gin router and handler for base64 tests.
func setupTestRouterForBase64(t *testing.T) (*gin.Engine, *Handler) {
	gin.SetMode(gin.TestMode)
	store, err := storage.NewFilesystemStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	cfg := &config.Config{
		BufferSize: 1024 * 1024, // 1MB
		DefaultTTL: 24 * time.Hour,
	}
	service := services.NewPasteService(store, cfg)
	handler := NewHandler(service, cfg)
	router := gin.New()
	return router, handler
}

// runBase64TestCase executes a single test case.
func runBase64TestCase(t *testing.T, handler *Handler, tt base64Test) {
	router := gin.New()
	if tt.useRoute {
		router.POST("/base64", func(c *gin.Context) {
			c.Request.Header.Set("X-Base64", "true")
			c.Next()
		}, handler.Upload)
	} else {
		router.POST("/", handler.Upload)
	}

	body := buildTestBody(tt)
	path := "/"
	if tt.useRoute {
		path = "/base64"
	}

	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	if tt.useBase64 && !tt.useRoute {
		req.Header.Set("X-Base64", "true")
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if tt.expectError {
		if w.Code == 200 {
			t.Errorf("Expected error but got success")
		}
		if tt.errorContains != "" && !strings.Contains(w.Body.String(), tt.errorContains) {
			t.Errorf("Expected error containing %q, got: %s", tt.errorContains, w.Body.String())
		}
	} else {
		if w.Code != 200 {
			t.Errorf("Expected success but got status %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "slug") {
			t.Errorf("Response missing slug: %s", w.Body.String())
		}
	}
}

// Valid cases for base64 decoding and upload flows. Keeping valid scenarios
// in a focused test reduces cyclomatic complexity compared to having all
// variants + error cases in one large test function.
func TestBase64Decoding_ValidCases(t *testing.T) {
	_, handler := setupTestRouterForBase64(t)

	tests := []base64Test{
		{
			name:        "Plain text upload - no encoding",
			content:     "Hello, World!",
			useBase64:   false,
			expectError: false,
		},
		{
			name:        "Base64 encoded via header",
			content:     "Hello, World!",
			useBase64:   true,
			useRoute:    false,
			expectError: false,
		},
		{
			name:        "Base64 encoded via route",
			content:     "Hello, World!",
			useBase64:   true,
			useRoute:    true,
			expectError: false,
		},
		{
			name:        "Base64 with shell script (WAF trigger)",
			content:     "#!/bin/bash\ncurl -X POST https://example.com --data-binary @-",
			useBase64:   true,
			expectError: false,
		},
		{
			name:        "Base64 with special characters",
			content:     "Line 1\nLine 2\tTab\r\nWindows Line",
			useBase64:   true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runBase64TestCase(t, handler, tt)
		})
	}
}

// Error cases intentionally separated into a focused test to keep each test's
// complexity within acceptable limits for static cyclomatic analysis.
func TestBase64Decoding_ErrorCases(t *testing.T) {
	_, handler := setupTestRouterForBase64(t)

	tests := []base64Test{
		{
			name:          "Invalid base64 content",
			content:       "This is not valid base64!!@#$%",
			useBase64:     true,
			expectError:   true,
			errorContains: "invalid base64 encoding",
		},
		{
			name:          "Empty decoded content",
			content:       "",
			useBase64:     true,
			expectError:   true,
			errorContains: "empty content", // Gets caught earlier in validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runBase64TestCase(t, handler, tt)
		})
	}
}

func TestBase64SizeLimits(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store, err := storage.NewFilesystemStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	cfg := &config.Config{
		BufferSize: 1024, // 1KB limit
		DefaultTTL: 24 * time.Hour,
	}

	service := services.NewPasteService(store, cfg)
	handler := NewHandler(service, cfg)

	router := gin.New()
	router.POST("/", handler.Upload)

	tests := []struct {
		name          string
		dataSize      int
		useBase64     bool
		expectError   bool
		errorContains string
	}{
		{
			name:        "Small content without base64",
			dataSize:    500,
			useBase64:   false,
			expectError: false,
		},
		{
			name:        "Small content with base64",
			dataSize:    500,
			useBase64:   true,
			expectError: false,
		},
		{
			name:        "Content at limit without base64",
			dataSize:    1024,
			useBase64:   false,
			expectError: false,
		},
		{
			name:        "Content at limit with base64 (decoded size)",
			dataSize:    1024,
			useBase64:   true,
			expectError: false,
		},
		{
			name:          "Content exceeds limit after decoding",
			dataSize:      2000,
			useBase64:     true,
			expectError:   true,
			errorContains: "content too large", // Caught during read, before decode
		},
		{
			name:          "Content exceeds limit without base64",
			dataSize:      2000,
			useBase64:     false,
			expectError:   true,
			errorContains: "content too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create content of specified size
			content := bytes.Repeat([]byte("A"), tt.dataSize)

			var body string
			if tt.useBase64 {
				body = base64.StdEncoding.EncodeToString(content)
			} else {
				body = string(content)
			}

			req := httptest.NewRequest("POST", "/", strings.NewReader(body))
			req.Header.Set("Content-Type", "text/plain")

			if tt.useBase64 {
				req.Header.Set("X-Base64", "true")
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tt.expectError {
				if w.Code == 200 {
					t.Errorf("Expected error but got success")
				}
				if tt.errorContains != "" && !strings.Contains(w.Body.String(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got: %s", tt.errorContains, w.Body.String())
				}
			} else {
				if w.Code != 200 {
					t.Errorf("Expected success but got status %d: %s", w.Code, w.Body.String())
				}
			}
		})
	}
}

func TestBase64MultipartUpload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store, err := storage.NewFilesystemStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	cfg := &config.Config{
		BufferSize: 1024 * 1024,
		DefaultTTL: 24 * time.Hour,
	}

	service := services.NewPasteService(store, cfg)
	handler := NewHandler(service, cfg)

	router := gin.New()
	router.POST("/", handler.Upload)

	// Test multipart upload with base64
	content := "Test file content with special chars: !@#$%^&*()"
	encodedContent := base64.StdEncoding.EncodeToString([]byte(content))

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	if _, err := io.WriteString(part, encodedContent); err != nil {
		t.Fatalf("Failed to write to form: %v", err)
	}

	writer.Close()

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Base64", "true")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected success but got status %d: %s", w.Code, w.Body.String())
	}
}

func TestBase64EncodingVariants(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store, err := storage.NewFilesystemStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	cfg := &config.Config{
		BufferSize: 1024 * 1024,
		DefaultTTL: 24 * time.Hour,
	}

	service := services.NewPasteService(store, cfg)
	handler := NewHandler(service, cfg)

	router := gin.New()
	router.POST("/", handler.Upload)

	content := "Test content!"

	tests := []struct {
		name     string
		encoding func([]byte) string
	}{
		{
			name:     "Standard base64",
			encoding: func(b []byte) string { return base64.StdEncoding.EncodeToString(b) },
		},
		{
			name:     "URL-safe base64",
			encoding: func(b []byte) string { return base64.URLEncoding.EncodeToString(b) },
		},
		{
			name:     "Raw standard base64 (no padding)",
			encoding: func(b []byte) string { return base64.RawStdEncoding.EncodeToString(b) },
		},
		{
			name:     "Raw URL-safe base64 (no padding)",
			encoding: func(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := tt.encoding([]byte(content))

			req := httptest.NewRequest("POST", "/", strings.NewReader(encoded))
			req.Header.Set("Content-Type", "text/plain")
			req.Header.Set("X-Base64", "true")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Errorf("Expected success for %s but got status %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

func TestXBurnHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create temp storage
	store, err := storage.NewFilesystemStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	cfg := &config.Config{
		BufferSize: 1024 * 1024,
		DefaultTTL: 24 * time.Hour,
	}

	service := services.NewPasteService(store, cfg)
	handler := NewHandler(service, cfg)

	router := gin.New()
	router.POST("/", handler.Upload)
	router.POST("/burn/", handler.UploadBurn)

	tests := []struct {
		name        string
		route       string
		xBurnHeader string
		expectBurn  bool
		description string
	}{
		{
			name:        "X-Burn: true on / route",
			route:       "/",
			xBurnHeader: "true",
			expectBurn:  true,
			description: "Should create burn paste via header",
		},
		{
			name:        "X-Burn: 1 on / route",
			route:       "/",
			xBurnHeader: "1",
			expectBurn:  true,
			description: "Should create burn paste with value '1'",
		},
		{
			name:        "X-Burn: yes on / route",
			route:       "/",
			xBurnHeader: "yes",
			expectBurn:  true,
			description: "Should create burn paste with value 'yes'",
		},
		{
			name:        "X-Burn: false on / route",
			route:       "/",
			xBurnHeader: "false",
			expectBurn:  false,
			description: "Should NOT create burn paste with value 'false'",
		},
		{
			name:        "X-Burn: 0 on / route",
			route:       "/",
			xBurnHeader: "0",
			expectBurn:  false,
			description: "Should NOT create burn paste with value '0'",
		},
		{
			name:        "No X-Burn header on / route",
			route:       "/",
			xBurnHeader: "",
			expectBurn:  false,
			description: "Should NOT create burn paste without header",
		},
		{
			name:        "/burn/ route without header",
			route:       "/burn/",
			xBurnHeader: "",
			expectBurn:  true,
			description: "Should create burn paste via route (backward compatibility)",
		},
		{
			name:        "X-Burn header overrides route",
			route:       "/burn/",
			xBurnHeader: "true",
			expectBurn:  true,
			description: "X-Burn header takes precedence over route",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := "Test burn content: " + tt.name

			req := httptest.NewRequest("POST", tt.route, strings.NewReader(content))
			req.Header.Set("Content-Type", "text/plain")
			if tt.xBurnHeader != "" {
				req.Header.Set("X-Burn", tt.xBurnHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			// Extract slug from response
			body := w.Body.String()
			if !strings.Contains(body, "http") {
				t.Fatalf("Response doesn't contain URL: %s", body)
			}

			// Get the slug (last part of URL after /)
			// Response format: http://localhost:8080/SLUG
			parts := strings.Split(strings.TrimSpace(body), "/")
			slug := strings.TrimSpace(parts[len(parts)-1])

			// Remove any JSON formatting artifacts
			slug = strings.Trim(slug, `"{}`)

			// Retrieve the paste metadata to check burn flag
			paste, err := store.Get(slug)
			if err != nil {
				t.Fatalf("Failed to retrieve paste (slug=%s): %v", slug, err)
			}

			if paste.BurnAfterRead != tt.expectBurn {
				t.Errorf("%s: Expected BurnAfterRead=%v, got %v", tt.description, tt.expectBurn, paste.BurnAfterRead)
			}
		})
	}
}

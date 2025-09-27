package handlers

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/config"
)

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

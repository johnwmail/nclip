package handlers

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/config"
)

func setupTestWebUIHandler(baseURL string) *WebUIHandler {
	cfg := &config.Config{
		URL:        baseURL,
		Version:    "1.0.0",
		BuildTime:  "2023-01-01T00:00:00Z",
		CommitHash: "abc123",
	}
	return NewWebUIHandler(cfg)
}

func TestWebUIHandler_NewWebUIHandler(t *testing.T) {
	cfg := &config.Config{
		URL:        "https://example.com",
		Version:    "1.0.0",
		BuildTime:  "2023-01-01T00:00:00Z",
		CommitHash: "abc123",
	}

	handler := NewWebUIHandler(cfg)

	if handler == nil {
		t.Fatal("Expected handler to be created, got nil")
	}

	if handler.config != cfg {
		t.Error("Expected config to be set correctly")
	}
}

func TestWebUIHandler_isHTTPS(t *testing.T) {
	handler := setupTestWebUIHandler("")

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
			name: "X-Forwarded-Protocol https",
			headers: map[string]string{
				"X-Forwarded-Protocol": "https",
			},
			hasTLS: false,
			want:   true,
		},
		{
			name: "X-Forwarded-Scheme https",
			headers: map[string]string{
				"X-Forwarded-Scheme": "https",
			},
			hasTLS: false,
			want:   true,
		},
		{
			name: "X-Scheme https",
			headers: map[string]string{
				"X-Scheme": "https",
			},
			hasTLS: false,
			want:   true,
		},
		{
			name: "X-Forwarded-Ssl on",
			headers: map[string]string{
				"X-Forwarded-Ssl": "on",
			},
			hasTLS: false,
			want:   true,
		},
		{
			name: "X-Forwarded-Https on",
			headers: map[string]string{
				"X-Forwarded-Https": "on",
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
			// Create test request
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
				t.Errorf("WebUIHandler.isHTTPS() = %v, want %v", got, tt.want)
			}
		})
	}
}

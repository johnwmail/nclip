package handlers

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestWebUIHandler_isCli(t *testing.T) {
	handler := setupTestWebUIHandler("")

	tests := []struct {
		name      string
		userAgent string
		accept    string
		want      bool
	}{
		{
			name:      "curl user agent",
			userAgent: "curl/7.81.0",
			accept:    "*/*",
			want:      true,
		},
		{
			name:      "curl user agent with Accept HTML should return HTML",
			userAgent: "curl/7.81.0",
			accept:    "text/html",
			want:      false,
		},
		{
			name:      "curl user agent with Accept HTML (case insensitive)",
			userAgent: "curl/7.81.0",
			accept:    "TEXT/HTML",
			want:      false,
		},
		{
			name:      "wget user agent",
			userAgent: "Wget/1.21.2",
			accept:    "*/*",
			want:      true,
		},
		{
			name:      "PowerShell user agent",
			userAgent: "Mozilla/5.0 (Windows NT; Windows NT 10.0; en-US) WindowsPowerShell/5.1.19041.1682",
			accept:    "*/*",
			want:      true,
		},
		{
			name:      "HTTPie user agent",
			userAgent: "HTTPie/3.2.1",
			accept:    "*/*",
			want:      true,
		},
		{
			name:      "Invoke-WebRequest user agent",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Microsoft Windows 10.0.19044; en-US) PowerShell/7.2.6",
			accept:    "*/*",
			want:      true,
		},
		{
			name:      "Invoke-RestMethod user agent",
			userAgent: "Mozilla/5.0 (Windows NT; Windows NT 10.0; en-US) Invoke-RestMethod",
			accept:    "*/*",
			want:      true,
		},
		{
			name:      "Chrome browser user agent",
			userAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36",
			accept:    "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			want:      false,
		},
		{
			name:      "Firefox browser user agent",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/115.0",
			accept:    "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			want:      false,
		},
		{
			name:      "Safari browser user agent",
			userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Safari/605.1.15",
			accept:    "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			want:      false,
		},
		{
			name:      "Empty user agent",
			userAgent: "",
			accept:    "*/*",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req, _ := http.NewRequest("GET", "/", nil)
			req.Header.Set("User-Agent", tt.userAgent)
			req.Header.Set("Accept", tt.accept)

			// Create test context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			got := handler.isCli(c)
			if got != tt.want {
				t.Errorf("WebUIHandler.isCli() with User-Agent=%q, Accept=%q = %v, want %v", tt.userAgent, tt.accept, got, tt.want)
			}
		})
	}
}

func TestWebUIHandler_Index_CLI(t *testing.T) {
	handler := setupTestWebUIHandler("http://localhost:8080")

	tests := []struct {
		name       string
		userAgent  string
		accept     string
		wantCLI    bool
		wantStatus int
	}{
		{
			name:       "curl request should return CLI usage",
			userAgent:  "curl/7.81.0",
			accept:     "*/*",
			wantCLI:    true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "wget request should return CLI usage",
			userAgent:  "Wget/1.21.2",
			accept:     "*/*",
			wantCLI:    true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "curl with Accept: text/html should return HTML",
			userAgent:  "curl/7.81.0",
			accept:     "text/html",
			wantCLI:    false,
			wantStatus: http.StatusOK,
		},
		{
			name:       "browser (Chrome) request should return HTML",
			userAgent:  "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36",
			accept:     "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			wantCLI:    false,
			wantStatus: http.StatusOK,
		},
		{
			name:       "browser (Firefox) request should return HTML",
			userAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/115.0",
			accept:     "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			wantCLI:    false,
			wantStatus: http.StatusOK,
		},
		{
			name:       "browser (Safari) request should return HTML",
			userAgent:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Safari/605.1.15",
			accept:     "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			wantCLI:    false,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test router
			gin.SetMode(gin.TestMode)
			router := gin.New()
			// Load HTML templates for browser tests
			router.LoadHTMLGlob("static/*.html")
			router.GET("/", handler.Index)

			// Create test request
			req, _ := http.NewRequest("GET", "/", nil)
			req.Header.Set("User-Agent", tt.userAgent)
			req.Header.Set("Accept", tt.accept)

			// Execute request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.wantStatus {
				t.Errorf("Status code = %d, want %d", w.Code, tt.wantStatus)
			}

			// Check content type and body
			contentType := w.Header().Get("Content-Type")
			body := w.Body.String()

			if tt.wantCLI {
				// For CLI, expect plain text
				if contentType != "text/plain; charset=utf-8" {
					t.Errorf("Content-Type = %q, want %q", contentType, "text/plain; charset=utf-8")
				}

				// Check that response contains expected CLI usage text
				if !containsAll(body, []string{"NCLIP", "Usage Examples", "curl", "echo"}) {
					t.Errorf("Response body does not contain expected CLI usage text")
				}
			} else {
				// For browser, expect HTML
				if contentType != "text/html; charset=utf-8" {
					t.Errorf("Content-Type = %q, want %q for browser request", contentType, "text/html; charset=utf-8")
				}

				// Check that response contains HTML elements
				if !containsAll(body, []string{"<!DOCTYPE html>", "<html", "NCLIP"}) {
					t.Errorf("Response body does not contain expected HTML content")
				}
			}
		})
	}
}

func TestWebUIHandler_Index_CLI_WithAuth(t *testing.T) {
	handler := &WebUIHandler{
		config: &config.Config{
			URL:        "http://localhost:8080",
			Version:    "1.0.0",
			UploadAuth: true,
		},
	}

	// Create test router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/", handler.Index)

	// Create test request mimicking curl
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("User-Agent", "curl/7.81.0")
	req.Header.Set("Accept", "*/*")

	// Execute request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Fatalf("Status code = %d, want %d", w.Code, http.StatusOK)
	}

	// For CLI + UploadAuth, expect plain text and auth examples
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want %q", contentType, "text/plain; charset=utf-8")
	}

	body := w.Body.String()
	if !containsAll(body, []string{"Support both Authorization and X-Api-Key", "X-Api-Key", "Authorization"}) {
		t.Fatalf("Response body does not contain expected API key authentication examples")
	}
}

// containsAll checks if the text contains all the given substrings
func containsAll(text string, substrings []string) bool {
	for _, s := range substrings {
		if !strings.Contains(text, s) {
			return false
		}
	}
	return true
}

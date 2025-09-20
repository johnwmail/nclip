package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/config"
	"github.com/johnwmail/nclip/handlers"
	"github.com/johnwmail/nclip/models"
)

// MockStore implements PasteStore for testing
type MockStore struct {
	pastes    map[string]*models.Paste
	readCount map[string]int
}

func NewMockStore() *MockStore {
	return &MockStore{
		pastes:    make(map[string]*models.Paste),
		readCount: make(map[string]int),
	}
}

func (m *MockStore) Store(paste *models.Paste) error {
	m.pastes[paste.ID] = paste
	return nil
}

func (m *MockStore) Get(id string) (*models.Paste, error) {
	paste, exists := m.pastes[id]
	if !exists {
		return nil, nil
	}
	// Check if expired
	if paste.IsExpired() {
		delete(m.pastes, id)
		return nil, nil
	}
	return paste, nil
}

func (m *MockStore) Delete(id string) error {
	delete(m.pastes, id)
	return nil
}

func (m *MockStore) IncrementReadCount(id string) error {
	if paste, exists := m.pastes[id]; exists {
		paste.ReadCount++
		m.readCount[id]++
	}
	return nil
}

func (m *MockStore) Close() error {
	return nil
}

func setupTestRouter() (*gin.Engine, *MockStore) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Port:          8080,
		SlugLength:    5,
		BufferSize:    1048576,
		DefaultTTL:    24 * time.Hour,
		EnableMetrics: true,
		EnableWebUI:   true,
	}

	store := NewMockStore()

	pasteHandler := handlers.NewPasteHandler(store, cfg)
	metaHandler := handlers.NewMetaHandler(store)
	systemHandler := handlers.NewSystemHandler()
	webuiHandler := handlers.NewWebUIHandler(cfg)

	router := gin.New()
	router.LoadHTMLGlob("static/*.html")
	router.Static("/static", "./static")

	// Routes
	if cfg.EnableWebUI {
		router.GET("/", webuiHandler.Index)
	}
	router.POST("/", pasteHandler.Upload)
	router.POST("/burn/", pasteHandler.UploadBurn)
	router.GET("/:slug", pasteHandler.View)
	router.GET("/raw/:slug", pasteHandler.Raw)
	router.GET("/api/v1/meta/:slug", metaHandler.GetMetadata)
	router.GET("/json/:slug", metaHandler.GetMetadata)
	router.GET("/health", systemHandler.Health)
	if cfg.EnableMetrics {
		router.GET("/metrics", systemHandler.Metrics)
	}

	return router, store
}

func TestHealthCheck(t *testing.T) {
	router, _ := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response) // Ignore error in test

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", response["status"])
	}
}

func TestUploadText(t *testing.T) {
	router, store := setupTestRouter()

	content := "Hello, World!"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", bytes.NewBufferString(content))
	req.Header.Set("Content-Type", "text/plain")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check if paste was stored
	if len(store.pastes) != 1 {
		t.Errorf("Expected 1 paste in store, got %d", len(store.pastes))
	}

	// Verify the response contains a URL
	responseBody := w.Body.String()
	if responseBody == "" {
		t.Error("Expected non-empty response body")
	}
}

func TestGetPaste(t *testing.T) {
	router, store := setupTestRouter()

	// First, create a paste
	paste := &models.Paste{
		ID:          "TEST2",
		CreatedAt:   time.Now(),
		Size:        5,
		ContentType: "text/plain",
		Content:     []byte("hello"),
	}
	store.Store(paste)

	// Now retrieve it
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/TEST2", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetRawPaste(t *testing.T) {
	router, store := setupTestRouter()

	content := []byte("raw content")
	paste := &models.Paste{
		ID:          "TEST3",
		CreatedAt:   time.Now(),
		Size:        int64(len(content)),
		ContentType: "text/plain",
		Content:     content,
	}
	store.Store(paste)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/raw/TEST3", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !bytes.Equal(w.Body.Bytes(), content) {
		t.Errorf("Expected body %s, got %s", content, w.Body.Bytes())
	}
}

func TestGetMetadata(t *testing.T) {
	router, store := setupTestRouter()

	paste := &models.Paste{
		ID:          "TEST4",
		CreatedAt:   time.Now(),
		Size:        10,
		ContentType: "text/plain",
		Content:     []byte("test content"),
	}
	store.Store(paste)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/meta/TEST4", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["id"] != "TEST4" {
		t.Errorf("Expected id 'TEST4', got %v", response["id"])
	}

	if response["size"] != float64(10) {
		t.Errorf("Expected size 10, got %v", response["size"])
	}
}

func TestGetMetadataAlias(t *testing.T) {
	router, store := setupTestRouter()

	paste := &models.Paste{
		ID:          "TEST5",
		CreatedAt:   time.Now(),
		Size:        15,
		ContentType: "text/plain",
		Content:     []byte("alias test content"),
	}
	store.Store(paste)

	// Test the /json/:slug alias
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/json/TEST5", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["id"] != "TEST5" {
		t.Errorf("Expected id 'TEST5', got %v", response["id"])
	}

	if response["size"] != float64(15) {
		t.Errorf("Expected size 15, got %v", response["size"])
	}

	// Verify both endpoints return the same data
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/v1/meta/TEST5", nil)
	router.ServeHTTP(w2, req2)

	if w.Body.String() != w2.Body.String() {
		t.Errorf("Alias route should return same data as original route")
	}
}

func TestBurnAfterRead(t *testing.T) {
	router, store := setupTestRouter()

	content := "burn this"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/burn/", bytes.NewBufferString(content))
	req.Header.Set("Content-Type", "text/plain")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Extract slug from response
	responseBody := w.Body.String()
	if responseBody == "" {
		t.Fatal("Expected non-empty response body")
	}

	// Find the paste in store to get its ID
	var pasteID string
	for id, paste := range store.pastes {
		if paste.BurnAfterRead {
			pasteID = id
			break
		}
	}

	if pasteID == "" {
		t.Fatal("Could not find burn-after-read paste")
	}

	// Read the paste once
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/"+pasteID, nil)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200 on first read, got %d", w2.Code)
	}

	// Try to read again - should be gone
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/"+pasteID, nil)
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 on second read, got %d", w3.Code)
	}
}

func TestNotFound(t *testing.T) {
	router, _ := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ABCDE", nil) // Valid slug format that doesn't exist
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestInvalidSlug(t *testing.T) {
	router, _ := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/invalid-slug!", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHTTPSOnly(t *testing.T) {
	// Test with HTTPS-only enabled
	store := NewMockStore()
	cfg := &config.Config{
		SlugLength:    5,
		BufferSize:    1048576,
		DefaultTTL:    24 * time.Hour,
		EnableMetrics: false,
		EnableWebUI:   true,
		HTTPSOnly:     true, // Enable HTTPS-only
		URL:           "",   // No explicit URL set
	}

	router := setupRouter(store, cfg)

	content := "test HTTPS-only"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", bytes.NewBufferString(content))
	req.Header.Set("Content-Type", "text/plain")
	req.Host = "example.com" // Set the host for URL generation
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Check that URL starts with https://
	url, ok := response["url"].(string)
	if !ok {
		t.Fatalf("URL not found in response")
	}

	expectedPrefix := "https://example.com/"
	if len(url) < len(expectedPrefix) || url[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Expected URL to start with %s, got %s", expectedPrefix, url)
	}
}

func TestHTTPSOnlyWithExplicitURL(t *testing.T) {
	// Test that explicit URL takes precedence over HTTPS-only
	store := NewMockStore()
	cfg := &config.Config{
		SlugLength:    5,
		BufferSize:    1048576,
		DefaultTTL:    24 * time.Hour,
		EnableMetrics: false,
		EnableWebUI:   true,
		HTTPSOnly:     true,                            // Enable HTTPS-only
		URL:           "http://custom-domain.com:8080", // Explicit URL with HTTP
	}

	router := setupRouter(store, cfg)

	content := "test explicit URL override"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", bytes.NewBufferString(content))
	req.Header.Set("Content-Type", "text/plain")
	req.Host = "example.com"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Check that URL uses the explicit URL (even though it's HTTP)
	url, ok := response["url"].(string)
	if !ok {
		t.Fatalf("URL not found in response")
	}

	expectedPrefix := "http://custom-domain.com:8080/"
	if len(url) < len(expectedPrefix) || url[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Expected URL to start with %s, got %s", expectedPrefix, url)
	}
}

func TestHTTPSOnlyDisabled(t *testing.T) {
	// Test with HTTPS-only disabled (default behavior)
	store := NewMockStore()
	cfg := &config.Config{
		SlugLength:    5,
		BufferSize:    1048576,
		DefaultTTL:    24 * time.Hour,
		EnableMetrics: false,
		EnableWebUI:   true,
		HTTPSOnly:     false, // Disable HTTPS-only
		URL:           "",    // No explicit URL set
	}

	router := setupRouter(store, cfg)

	content := "test HTTP default"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", bytes.NewBufferString(content))
	req.Header.Set("Content-Type", "text/plain")
	req.Host = "example.com"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Check that URL starts with http:// (default behavior)
	url, ok := response["url"].(string)
	if !ok {
		t.Fatalf("URL not found in response")
	}

	expectedPrefix := "http://example.com/"
	if len(url) < len(expectedPrefix) || url[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Expected URL to start with %s, got %s", expectedPrefix, url)
	}
}

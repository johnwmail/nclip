package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/config"
	"github.com/johnwmail/nclip/handlers"
	"github.com/johnwmail/nclip/handlers/retrieval"
	"github.com/johnwmail/nclip/handlers/upload"
	"github.com/johnwmail/nclip/internal/services"
	"github.com/johnwmail/nclip/models"
)

// MockStore implements PasteStore for testing
type MockStore struct {
	pastes    map[string]*models.Paste
	readCount map[string]int
	// For content storage
	content map[string][]byte
	dataDir string
}

func NewMockStore(dataDir string) *MockStore {
	return &MockStore{
		pastes:    make(map[string]*models.Paste),
		readCount: make(map[string]int),
		content:   make(map[string][]byte),
		dataDir:   dataDir,
	}
}

// cleanupTestData removes the test data directory and all its contents.
// Should be called with defer in each test that creates files.
func cleanupTestData(dataDir string) {
	if dataDir == "" {
		dataDir = "./data"
	}
	_ = os.RemoveAll(dataDir)
}

// StoreContent saves the raw content for a paste
func (m *MockStore) StoreContent(id string, content []byte) error {
	m.content[id] = content
	// Also write the content to disk inside NCLIP_DATA_DIR so handlers that
	// perform rename/mv inside that directory succeed during tests.
	dataDir := m.dataDir
	if dataDir == "" {
		dataDir = "./data"
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	contentPath := filepath.Join(dataDir, id)
	if err := os.WriteFile(contentPath, content, 0o644); err != nil {
		return err
	}
	return nil
}

// GetContent retrieves the raw content for a paste
func (m *MockStore) GetContent(id string) ([]byte, error) {
	c, ok := m.content[id]
	if !ok {
		return nil, nil
	}
	return c, nil
}

// GetContentPrefix reads up to n bytes of content for tests and also from the
// on-disk copy so handlers that do rename/mv in data dir can read preview.
func (m *MockStore) GetContentPrefix(id string, n int64) ([]byte, error) {
	if c, ok := m.content[id]; ok {
		if int64(len(c)) <= n {
			return c, nil
		}
		return c[:n], nil
	}
	// Fallback to disk using the configured dataDir on the mock
	dataDir := m.dataDir
	if dataDir == "" {
		dataDir = "./data"
	}
	contentPath := filepath.Join(dataDir, id)
	f, err := os.Open(contentPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, n)
	read, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return buf[:read], nil
}

func (m *MockStore) Store(paste *models.Paste) error {
	m.pastes[paste.ID] = paste
	if paste.Content != nil {
		m.StoreContent(paste.ID, paste.Content)
	}
	// Also write metadata file to disk to emulate FilesystemStore behavior for tests.
	dataDir := m.dataDir
	if dataDir == "" {
		dataDir = "./data"
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	metaPath := filepath.Join(dataDir, paste.ID+".json")
	b, _ := json.MarshalIndent(paste, "", "  ")
	_ = os.WriteFile(metaPath, b, 0o644)
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

func (m *MockStore) Exists(id string) (bool, error) {
	_, exists := m.pastes[id]
	return exists, nil
}

func (m *MockStore) Delete(id string) error {
	delete(m.pastes, id)
	dataDir := m.dataDir
	if dataDir == "" {
		dataDir = "./data"
	}
	_ = os.Remove(filepath.Join(dataDir, id))
	_ = os.Remove(filepath.Join(dataDir, id+".json"))
	delete(m.content, id)
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

// StatContent reports whether content exists on disk (from dataDir) and its size.
func (m *MockStore) StatContent(id string) (bool, int64, error) {
	dataDir := m.dataDir
	if dataDir == "" {
		dataDir = "./data"
	}
	contentPath := filepath.Join(dataDir, id)
	st, err := os.Stat(contentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, 0, nil
		}
		return false, 0, err
	}
	return true, st.Size(), nil
}

func setupTestRouter() (*gin.Engine, *MockStore) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Port:       8080,
		SlugLength: 5,
		BufferSize: 5 * 1024 * 1024,
		DefaultTTL: 24 * time.Hour,
	}

	store := NewMockStore(cfg.DataDir)

	pasteService := services.NewPasteService(store, cfg)
	uploadHandler := upload.NewHandler(pasteService, cfg)
	retrievalHandler := retrieval.NewHandler(pasteService, store, cfg)
	metaHandler := handlers.NewMetaHandler(store)
	systemHandler := handlers.NewSystemHandler()
	webuiHandler := handlers.NewWebUIHandler(cfg)

	router := gin.New()
	router.LoadHTMLGlob("static/*.html")
	router.Static("/static", "./static")

	// Routes (WebUI always enabled)
	router.GET("/", webuiHandler.Index)
	router.POST("/", uploadHandler.Upload)
	router.POST("/burn/", uploadHandler.UploadBurn)
	router.GET("/:slug", retrievalHandler.View)
	router.GET("/raw/:slug", retrievalHandler.Raw)
	router.GET("/api/v1/meta/:slug", metaHandler.GetMetadata)
	router.GET("/json/:slug", metaHandler.GetMetadata)
	router.GET("/health", systemHandler.Health)

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

func TestMetricsEndpointRemoved(t *testing.T) {
	router, _ := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/metrics", nil)
	router.ServeHTTP(w, req)

	// The /metrics endpoint should no longer exist as a specific route.
	// It will be caught by the /:slug route and return 400 (invalid slug format)
	// or 404 (slug not found). Either way, it's not a 200 with metrics data.
	if w.Code == http.StatusOK {
		t.Errorf("Metrics endpoint should NOT return 200 (metrics removed), got %d", w.Code)
	}

	// Verify it's not returning Prometheus metrics format
	body := w.Body.String()
	if bytes.Contains(w.Body.Bytes(), []byte("# HELP")) ||
		bytes.Contains(w.Body.Bytes(), []byte("# TYPE")) ||
		bytes.Contains(w.Body.Bytes(), []byte("prometheus")) {
		t.Errorf("Response should not contain Prometheus metrics format, but it does: %s", body)
	}
}

func TestUploadText(t *testing.T) {
	router, store := setupTestRouter()
	defer cleanupTestData(store.dataDir)

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
	defer cleanupTestData(store.dataDir)

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
	defer cleanupTestData(store.dataDir)

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
	defer cleanupTestData(store.dataDir)

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
	defer cleanupTestData(store.dataDir)

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
	defer cleanupTestData(store.dataDir)

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
	router, store := setupTestRouter()
	defer cleanupTestData(store.dataDir)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ABCDE", nil) // Valid slug format that doesn't exist
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestInvalidSlug(t *testing.T) {
	router, store := setupTestRouter()
	defer cleanupTestData(store.dataDir)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/invalid-slug!", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestAPIKeyAuth tests the API key authentication middleware
func TestAPIKeyAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		apiKeys        string
		authHeader     string
		apiKeyHeader   string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "valid Bearer token",
			apiKeys:        "key1,key2,key3",
			authHeader:     "Bearer key2",
			apiKeyHeader:   "",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "valid X-Api-Key header",
			apiKeys:        "secret123,secret456",
			authHeader:     "",
			apiKeyHeader:   "secret123",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "invalid Bearer token",
			apiKeys:        "key1,key2",
			authHeader:     "Bearer wrongkey",
			apiKeyHeader:   "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "unauthorized",
		},
		{
			name:           "invalid X-Api-Key",
			apiKeys:        "key1,key2",
			authHeader:     "",
			apiKeyHeader:   "badkey",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "unauthorized",
		},
		{
			name:           "missing auth headers",
			apiKeys:        "key1,key2",
			authHeader:     "",
			apiKeyHeader:   "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "missing api key",
		},
		{
			name:           "Bearer token with extra spaces",
			apiKeys:        "mykey",
			authHeader:     "Bearer   mykey  ",
			apiKeyHeader:   "",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "X-Api-Key with extra spaces",
			apiKeys:        "mykey",
			authHeader:     "",
			apiKeyHeader:   "  mykey  ",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "Bearer takes precedence over X-Api-Key",
			apiKeys:        "key1,key2",
			authHeader:     "Bearer key1",
			apiKeyHeader:   "wrongkey",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "case-insensitive Bearer prefix",
			apiKeys:        "mykey",
			authHeader:     "bearer mykey",
			apiKeyHeader:   "",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "malformed Authorization header",
			apiKeys:        "key1",
			authHeader:     "InvalidFormat key1",
			apiKeyHeader:   "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "missing api key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config with API keys
			cfg := &config.Config{
				APIKeys: tt.apiKeys,
			}

			// Create test router with auth middleware
			router := gin.New()
			authMiddleware := apiKeyAuth(cfg)
			router.POST("/test", authMiddleware, func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			// Create request
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString("test"))

			// Set headers
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			if tt.apiKeyHeader != "" {
				req.Header.Set("X-Api-Key", tt.apiKeyHeader)
			}

			// Execute request
			router.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check error message if unauthorized
			if tt.expectedStatus == http.StatusUnauthorized {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				if response["error"] != tt.expectedError {
					t.Errorf("Expected error '%s', got '%v'", tt.expectedError, response["error"])
				}
			}
		})
	}
}

// TestAPIKeyAuthEmptyKeys tests behavior when no API keys are configured
func TestAPIKeyAuthEmptyKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		APIKeys: "",
	}

	router := gin.New()
	authMiddleware := apiKeyAuth(cfg)
	router.POST("/test", authMiddleware, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString("test"))
	req.Header.Set("Authorization", "Bearer anykey")

	router.ServeHTTP(w, req)

	// Should fail because no keys are configured
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

// TestAPIKeyAuthWhitespaceInConfig tests handling of whitespace in API keys
func TestAPIKeyAuthWhitespaceInConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		APIKeys: " key1 , key2 , key3 ",
	}

	router := gin.New()
	authMiddleware := apiKeyAuth(cfg)
	router.POST("/test", authMiddleware, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	tests := []struct {
		key            string
		expectedStatus int
	}{
		{"key1", http.StatusOK},
		{"key2", http.StatusOK},
		{"key3", http.StatusOK},
		{" key1 ", http.StatusOK}, // Spaces are trimmed from request headers
		{"wrongkey", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString("test"))
			req.Header.Set("X-Api-Key", tt.key)

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("For key '%s', expected status %d, got %d", tt.key, tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestUploadAuthEnforced verifies that when cfg.UploadAuth is true,
// the router applies the apiKeyAuth middleware to upload endpoints.
func TestUploadAuthEnforced(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		APIKeys:    "testkey",
		UploadAuth: true,
		Port:       8080,
		SlugLength: 5,
		BufferSize: 5 * 1024 * 1024,
		DefaultTTL: 24 * time.Hour,
	}

	store := NewMockStore(cfg.DataDir)
	defer cleanupTestData(store.dataDir)

	// Use the real setupRouter so middleware wiring is exercised
	router := setupRouter(store, cfg)

	// POST without any auth should be rejected
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", bytes.NewBufferString("hello"))
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 when UploadAuth enabled and no key provided, got %d (body: %s)", w.Code, w.Body.String())
	}

	// POST with valid Authorization: Bearer header should succeed
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/", bytes.NewBufferString("hello"))
	req2.Header.Set("Authorization", "Bearer testkey")
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("Expected 200 when valid key provided, got %d (body: %s)", w2.Code, w2.Body.String())
	}
}

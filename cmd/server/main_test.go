package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/johnwmail/nclip/internal/config"
	"github.com/johnwmail/nclip/internal/server"
	"github.com/johnwmail/nclip/internal/storage"
)

func TestHTTPPasteCreation(t *testing.T) {
	// Create temporary storage
	tempDir := t.TempDir()
	store, err := storage.NewFilesystemStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close storage: %v", err)
		}
	}()

	// Create test config
	cfg := config.DefaultConfig()
	cfg.OutputDir = tempDir

	// Create test logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create HTTP server
	httpServer := server.NewHTTPServer(cfg, store, logger)

	// Test paste creation
	content := "Hello, World!\nThis is a test paste."
	req := httptest.NewRequest("POST", "/", strings.NewReader(content))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Filename", "test.txt")

	w := httptest.NewRecorder()
	handler := httpServer.TestHandler()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Check response contains URL
	response := w.Body.String()
	if !strings.Contains(response, "http") {
		t.Fatalf("Response should contain URL, got: %s", response)
	}

	// Extract paste ID from response
	parts := strings.Split(strings.TrimSpace(response), "/")
	if len(parts) == 0 {
		t.Fatalf("Could not extract paste ID from response: %s", response)
	}
	pasteID := parts[len(parts)-1]

	// Verify paste was stored
	paste, err := store.Get(pasteID)
	if err != nil {
		t.Fatalf("Failed to retrieve paste: %v", err)
	}

	if strings.TrimSpace(string(paste.Content)) != strings.TrimSpace(content) {
		t.Fatalf("Content mismatch. Expected: %q, got: %q", content, string(paste.Content))
	}

	if paste.Filename != "test.txt" {
		t.Fatalf("Filename mismatch. Expected: test.txt, got: %s", paste.Filename)
	}
}

func TestStorageBasics(t *testing.T) {
	// Create temporary storage
	tempDir := t.TempDir()
	store, err := storage.NewFilesystemStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close storage: %v", err)
		}
	}()

	// Test paste creation
	paste := &storage.Paste{
		ID:          "test123",
		Content:     []byte("Hello, World!"),
		ContentType: "text/plain",
		CreatedAt:   time.Now(),
		ClientIP:    "127.0.0.1",
		Size:        13,
	}

	// Store paste
	if err := store.Store(paste); err != nil {
		t.Fatalf("Failed to store paste: %v", err)
	}

	// Check exists
	if !store.Exists("test123") {
		t.Fatal("Paste should exist")
	}

	// Retrieve paste
	retrieved, err := store.Get("test123")
	if err != nil {
		t.Fatalf("Failed to retrieve paste: %v", err)
	}

	if string(retrieved.Content) != "Hello, World!" {
		t.Fatalf("Content mismatch. Expected: Hello, World!, got: %s", string(retrieved.Content))
	}

	// Test stats
	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.TotalPastes != 1 {
		t.Fatalf("Expected 1 paste, got %d", stats.TotalPastes)
	}
}

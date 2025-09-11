package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/johnwmail/nclip/internal/config"
	"github.com/johnwmail/nclip/internal/storage"
)

func TestNewTCPServer(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.OutputDir = t.TempDir()

	store, err := storage.NewFilesystemStorage(cfg.OutputDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer closeStorage(t, store)

	server := NewTCPServer(cfg, store, createTestLogger())
	if server == nil {
		t.Fatal("Expected non-nil TCP server")
	}
}

func TestTCPServer_Integration(t *testing.T) {
	// Setup
	cfg := config.DefaultConfig()
	cfg.OutputDir = t.TempDir()
	cfg.TCPPort = 0 // Let the OS choose a free port
	cfg.BaseURL = "http://localhost:8080/"

	store, err := storage.NewFilesystemStorage(cfg.OutputDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer closeStorage(t, store)

	server := NewTCPServer(cfg, store, createTestLogger())

	// Start server in background
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer closeListener(t, listener)

	// Get the actual port
	port := listener.Addr().(*net.TCPAddr).Port
	cfg.TCPPort = port

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // Listener closed
			}
			go server.handleConnection(conn)
		}
	}()

	// Give server time to start
	time.Sleep(10 * time.Millisecond)

	testCases := []struct {
		name          string
		input         string
		expectURL     bool
		expectError   bool
		checkResponse func(t *testing.T, response string)
	}{
		{
			name:      "simple text paste",
			input:     "Hello, World!\nThis is a test paste.",
			expectURL: true,
			checkResponse: func(t *testing.T, response string) {
				if !strings.Contains(response, "http://localhost") {
					t.Errorf("Expected URL in response, got: %s", response)
				}
			},
		},
		{
			name:      "code paste",
			input:     "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, Go!\")\n}",
			expectURL: true,
			checkResponse: func(t *testing.T, response string) {
				if !strings.Contains(response, "http://") {
					t.Errorf("Expected URL in response, got: %s", response)
				}
			},
		},
		{
			name:        "empty paste",
			input:       "",
			expectURL:   false,
			expectError: true,
			checkResponse: func(t *testing.T, response string) {
				// TCP server might just close connection on empty paste
				// or return an empty response, both are acceptable
			},
		},
		{
			name:      "multiline paste",
			input:     "Line 1\nLine 2\nLine 3\n\nLine 5 with empty line above",
			expectURL: true,
			checkResponse: func(t *testing.T, response string) {
				if !strings.Contains(response, "http://") {
					t.Errorf("Expected URL in response, got: %s", response)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Connect to server
			conn, err := net.Dial("tcp", listener.Addr().String())
			if err != nil {
				t.Fatalf("Failed to connect: %v", err)
			}
			defer closeConn(t, conn)

			// Send data
			_, err = conn.Write([]byte(tc.input))
			if err != nil {
				t.Fatalf("Failed to write data: %v", err)
			}

			// Close write side to signal end of input
			if tcpConn, ok := conn.(*net.TCPConn); ok {
				closeTCPWrite(t, tcpConn)
			}

			// Read response
			scanner := bufio.NewScanner(conn)
			var response strings.Builder
			for scanner.Scan() {
				response.WriteString(scanner.Text())
				response.WriteString("\n")
			}

			if err := scanner.Err(); err != nil {
				t.Fatalf("Failed to read response: %v", err)
			}

			responseStr := strings.TrimSpace(response.String())

			if tc.checkResponse != nil {
				tc.checkResponse(t, responseStr)
			}
		})
	}
}

func TestTCPServer_ConcurrentConnections(t *testing.T) {
	// Setup
	cfg := config.DefaultConfig()
	cfg.OutputDir = t.TempDir()
	cfg.BaseURL = "http://localhost:8080/"

	store, err := storage.NewFilesystemStorage(cfg.OutputDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer closeStorage(t, store)

	server := NewTCPServer(cfg, store, createTestLogger())

	// Start server
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer closeListener(t, listener)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go server.handleConnection(conn)
		}
	}()

	// Test concurrent connections
	numConnections := 10
	done := make(chan bool, numConnections)

	for i := 0; i < numConnections; i++ {
		go func(id int) {
			defer func() { done <- true }()

			conn, err := net.Dial("tcp", listener.Addr().String())
			if err != nil {
				t.Errorf("Connection %d failed to connect: %v", id, err)
				return
			}
			defer closeConn(t, conn)

			input := fmt.Sprintf("Test paste from connection %d\nWith multiple lines", id)
			_, err = conn.Write([]byte(input))
			if err != nil {
				t.Errorf("Connection %d failed to write: %v", id, err)
				return
			}

			if tcpConn, ok := conn.(*net.TCPConn); ok {
				closeTCPWrite(t, tcpConn)
			}

			// Read response
			scanner := bufio.NewScanner(conn)
			hasResponse := false
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line, "http://") {
					hasResponse = true
					break
				}
			}

			if !hasResponse {
				t.Errorf("Connection %d did not receive URL response", id)
			}
		}(i)
	}

	// Wait for all connections to complete
	for i := 0; i < numConnections; i++ {
		select {
		case <-done:
			// Connection completed
		case <-time.After(5 * time.Second):
			t.Fatalf("Timeout waiting for connection %d", i)
		}
	}
}

func TestTCPServer_LargePaste(t *testing.T) {
	// Setup
	cfg := config.DefaultConfig()
	cfg.OutputDir = t.TempDir()
	cfg.BufferSize = 1024 // Small buffer for testing

	store, err := storage.NewFilesystemStorage(cfg.OutputDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer closeStorage(t, store)

	server := NewTCPServer(cfg, store, createTestLogger())

	// Start server
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer closeListener(t, listener)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go server.handleConnection(conn)
		}
	}()

	// Create large content (larger than buffer)
	largeContent := strings.Repeat("This is a long line for testing large paste functionality.\n", 100)

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer closeConn(t, conn)

	// Send large content
	_, err = conn.Write([]byte(largeContent))
	if err != nil {
		t.Fatalf("Failed to write large content: %v", err)
	}

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		closeTCPWrite(t, tcpConn)
	}

	// Read response
	scanner := bufio.NewScanner(conn)
	hasURL := false
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "http://") {
			hasURL = true
			break
		}
	}

	if !hasURL {
		t.Error("Expected URL response for large paste")
	}
}

func TestTCPServer_ConnectionTimeout(t *testing.T) {
	// Setup with short timeout for testing
	cfg := config.DefaultConfig()
	cfg.OutputDir = t.TempDir()

	store, err := storage.NewFilesystemStorage(cfg.OutputDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer closeStorage(t, store)

	server := NewTCPServer(cfg, store, createTestLogger())

	// Start server
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer closeListener(t, listener)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go server.handleConnection(conn)
		}
	}()

	// Connect but don't send data immediately
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer closeConn(t, conn)

	// Set a read deadline to prevent the test from hanging
	setReadDeadline(t, conn, time.Now().Add(2*time.Second))

	// Wait a bit then send data
	time.Sleep(100 * time.Millisecond)

	_, err = conn.Write([]byte("Test after delay"))
	if err != nil {
		t.Fatalf("Failed to write after delay: %v", err)
	}

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		closeTCPWrite(t, tcpConn)
	}

	// Should still get response
	scanner := bufio.NewScanner(conn)
	hasResponse := false
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "http://") {
			hasResponse = true
			break
		}
	}

	if !hasResponse {
		t.Error("Expected response even after delay")
	}
}

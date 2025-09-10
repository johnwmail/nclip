package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/johnwmail/nclip/internal/config"
	"github.com/johnwmail/nclip/internal/slug"
	"github.com/johnwmail/nclip/internal/storage"
)

// TCPServer handles netcat connections
type TCPServer struct {
	config   *config.Config
	storage  storage.Storage
	slugGen  *slug.Generator
	listener net.Listener
	logger   *slog.Logger
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewTCPServer creates a new TCP server
func NewTCPServer(cfg *config.Config, storage storage.Storage, logger *slog.Logger) *TCPServer {
	ctx, cancel := context.WithCancel(context.Background())

	return &TCPServer{
		config:  cfg,
		storage: storage,
		slugGen: slug.New(cfg.SlugLength),
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start starts the TCP server
func (s *TCPServer) Start() error {
	addr := fmt.Sprintf(":%d", s.config.TCPPort)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.listener = listener
	s.logger.Info("TCP server started", "address", addr)

	go s.acceptConnections()

	return nil
}

// Stop stops the TCP server
func (s *TCPServer) Stop() error {
	s.cancel()

	if s.listener != nil {
		return s.listener.Close()
	}

	return nil
}

// acceptConnections accepts and handles incoming connections
func (s *TCPServer) acceptConnections() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			if s.ctx.Err() != nil {
				return // Server is shutting down
			}
			s.logger.Error("Failed to accept connection", "error", err)
			continue
		}

		go s.handleConnection(conn)
	}
}

// handleConnection handles a single TCP connection
func (s *TCPServer) handleConnection(conn net.Conn) {
	defer func() {
		if err := conn.Close(); err != nil {
			s.logger.Debug("Failed to close connection", "error", err)
		}
	}()

	// Set read timeout
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		s.logger.Error("Failed to set read deadline", "error", err)
		return
	}

	clientAddr := conn.RemoteAddr().String()
	clientIP := strings.Split(clientAddr, ":")[0]

	s.logger.Info("New TCP connection", "client", clientAddr)

	// Helper function for writing responses
	writeResponse := func(message string) {
		if _, err := conn.Write([]byte(message)); err != nil {
			s.logger.Debug("Failed to write response to client", "client", clientAddr, "error", err)
		}
	}

	// Read data from connection
	buffer := make([]byte, s.config.BufferSize)
	n, err := conn.Read(buffer)
	if err != nil {
		if err != io.EOF {
			s.logger.Error("Failed to read from connection", "client", clientAddr, "error", err)
		}
		return
	}

	if n == 0 {
		s.logger.Warn("Empty paste received", "client", clientAddr)
		writeResponse("Error: Empty paste\n")
		return
	}

	// Trim buffer to actual content
	content := buffer[:n]

	// Remove trailing null bytes and whitespace
	content = trimNullBytes(content)

	if len(content) == 0 {
		s.logger.Warn("Empty paste after cleanup", "client", clientAddr)
		writeResponse("Error: Empty paste\n")
		return
	}

	// Generate unique slug
	slugStr, err := s.slugGen.GenerateWithCollisionCheck(s.storage.Exists)
	if err != nil {
		s.logger.Error("Failed to generate slug", "error", err)
		writeResponse("Error: Could not generate paste ID\n")
		return
	}

	// Create paste
	paste := &storage.Paste{
		ID:          slugStr,
		Content:     content,
		ContentType: "text/plain",
		CreatedAt:   time.Now(),
		ClientIP:    clientIP,
		Size:        int64(len(content)),
	}

	// Set expiration if configured
	if expiration := s.config.GetExpiration(); expiration > 0 {
		expiresAt := time.Now().Add(expiration)
		paste.ExpiresAt = &expiresAt
	}

	// Store paste
	if err := s.storage.Store(paste); err != nil {
		s.logger.Error("Failed to store paste", "slug", slugStr, "error", err)
		writeResponse("Error: Could not save paste\n")
		return
	}

	// Generate URL
	url := fmt.Sprintf("%s/%s\n", s.config.GetBaseURL(), slugStr)

	// Send response
	writeResponse(url)

	s.logger.Info("Paste created via TCP",
		"slug", slugStr,
		"client", clientAddr,
		"size", len(content))
}

// trimNullBytes removes null bytes and trailing whitespace
func trimNullBytes(data []byte) []byte {
	// Find the last non-null byte
	end := len(data)
	for end > 0 && data[end-1] == 0 {
		end--
	}

	// Trim trailing whitespace but preserve some newlines for readability
	trimmed := strings.TrimRightFunc(string(data[:end]), func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\r'
	})

	return []byte(trimmed)
}

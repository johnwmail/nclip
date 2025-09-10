package server

import (
	"net"
	"testing"
	"time"

	"github.com/johnwmail/nclip/internal/storage"
)

// Helper functions for proper resource cleanup in tests

func closeStorage(t *testing.T, store storage.Storage) {
	if err := store.Close(); err != nil {
		t.Logf("Failed to close storage: %v", err)
	}
}

func closeListener(t *testing.T, listener net.Listener) {
	if err := listener.Close(); err != nil {
		t.Logf("Failed to close listener: %v", err)
	}
}

func closeConn(t *testing.T, conn net.Conn) {
	if err := conn.Close(); err != nil {
		t.Logf("Failed to close connection: %v", err)
	}
}

func closeTCPWrite(t *testing.T, conn *net.TCPConn) {
	if err := conn.CloseWrite(); err != nil {
		t.Logf("Failed to close TCP write: %v", err)
	}
}

func setReadDeadline(t *testing.T, conn net.Conn, deadline time.Time) {
	if err := conn.SetReadDeadline(deadline); err != nil {
		t.Logf("Failed to set read deadline: %v", err)
	}
}

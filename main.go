package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/johnwmail/nclip/internal/config"
	"github.com/johnwmail/nclip/internal/server"
	"github.com/johnwmail/nclip/internal/storage"
)

var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// Detect environment - if AWS_LAMBDA_RUNTIME_API is set, we're running in Lambda
	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" || os.Getenv("_LAMBDA_SERVER_PORT") != "" {
		runAsLambda()
	} else {
		runAsServer()
	}
}

// runAsLambda runs the application as an AWS Lambda function
func runAsLambda() {
	// Lambda deployment only supports DynamoDB storage
	// The actual handler should be implemented in a shared package, e.g. internal/server or internal/storage
	// For now, panic to indicate missing handler implementation
	panic("Lambda handler not implemented: please refactor to use shared logic")
}

// runAsServer runs the application as a traditional HTTP server
func runAsServer() {
	// Load configuration
	cfg, err := config.LoadFromFlags()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	logger := setupLogging(cfg)

	logger.Info("Starting nclip",
		"version", version,
		"build_time", buildTime,
		"git_commit", gitCommit,
		"base_url", cfg.BaseURL,
		"tcp_port", cfg.TCPPort,
		"http_port", cfg.HTTPPort)

	// Initialize storage
	store, err := storage.NewStorage(cfg, logger)
	if err != nil {
		logger.Error("Failed to initialize storage", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			logger.Error("Failed to close storage", "error", err)
		}
	}()

	// Create servers
	tcpServer := server.NewTCPServer(cfg, store, logger)
	httpServer := server.NewHTTPServer(cfg, store, logger)

	// Start servers
	if err := tcpServer.Start(); err != nil {
		logger.Error("Failed to start TCP server", "error", err)
		os.Exit(1)
	}

	if err := httpServer.Start(); err != nil {
		logger.Error("Failed to start HTTP server", "error", err)
		os.Exit(1)
	}

	logger.Info("Servers started successfully")
	logger.Info("Ready to accept connections",
		"netcat_usage", fmt.Sprintf("echo 'test' | nc %s %d", getHostFromURL(cfg.BaseURL), cfg.TCPPort),
		"curl_usage", fmt.Sprintf("echo 'test' | curl -d @- %s", cfg.GetBaseURL()),
		"web_interface", cfg.GetBaseURL())

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutdown signal received, stopping servers...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop servers (we'll use ctx for timeout if needed later)
	_ = ctx // Mark as used for now
	if err := tcpServer.Stop(); err != nil {
		logger.Error("Error stopping TCP server", "error", err)
	}

	if err := httpServer.Stop(); err != nil {
		logger.Error("Error stopping HTTP server", "error", err)
	}

	// Run cleanup
	logger.Info("Running cleanup...")
	if err := store.Cleanup(); err != nil {
		logger.Error("Error during cleanup", "error", err)
	}

	logger.Info("Shutdown complete")
}

// setupLogging configures structured logging
func setupLogging(cfg *config.Config) *slog.Logger {
	var handler slog.Handler

	// Configure log level
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Configure output
	if cfg.LogFile != "" {
		// Log to file
		file, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
			os.Exit(1)
		}
		handler = slog.NewJSONHandler(file, opts)
	} else {
		// Log to stderr
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	return slog.New(handler)
}

// getHostFromURL extracts the hostname from a URL for netcat examples
func getHostFromURL(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		// Fallback to localhost if URL parsing fails
		return "localhost"
	}

	// Return just the hostname without port
	if parsedURL.Hostname() != "" {
		return parsedURL.Hostname()
	}

	return "localhost"
}

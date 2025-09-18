package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/config"
	"github.com/johnwmail/nclip/handlers"
	"github.com/johnwmail/nclip/storage"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize storage backend
	var store storage.PasteStore
	var err error

	switch cfg.StorageType {
	case "mongodb":
		store, err = storage.NewMongoStore(cfg.MongoURL, "nclip")
		if err != nil {
			log.Fatalf("Failed to initialize MongoDB storage: %v", err)
		}
	case "dynamodb":
		store, err = storage.NewDynamoStore(cfg.DynamoTable, cfg.DynamoRegion)
		if err != nil {
			log.Fatalf("Failed to initialize DynamoDB storage: %v", err)
		}
	default:
		log.Fatalf("Unknown storage type: %s", cfg.StorageType)
	}

	// Ensure cleanup on exit
	defer func() {
		if err := store.Close(); err != nil {
			log.Printf("Error closing storage: %v", err)
		}
	}()

	// Initialize handlers
	pasteHandler := handlers.NewPasteHandler(store, cfg)
	metaHandler := handlers.NewMetaHandler(store)
	systemHandler := handlers.NewSystemHandler()
	webuiHandler := handlers.NewWebUIHandler(cfg)

	// Create Gin router
	router := gin.New()

	// Add logging middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Load HTML templates
	router.LoadHTMLGlob("static/*.html")

	// Serve static files
	router.Static("/static", "./static")

	// Web UI routes
	if cfg.EnableWebUI {
		router.GET("/", webuiHandler.Index)
	}

	// Core API routes
	router.POST("/", pasteHandler.Upload)
	router.POST("/burn/", pasteHandler.UploadBurn)
	router.GET("/:slug", pasteHandler.View)
	router.GET("/raw/:slug", pasteHandler.Raw)

	// Metadata API
	router.GET("/api/v1/meta/:slug", metaHandler.GetMetadata)

	// Alias for metadata API (shortcut)
	router.GET("/json/:slug", metaHandler.GetMetadata)

	// System routes
	router.GET("/health", systemHandler.Health)
	if cfg.EnableMetrics {
		router.GET("/metrics", systemHandler.Metrics)
	}

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting nclip server on port %d", cfg.Port)
		log.Printf("Storage backend: %s", cfg.StorageType)
		log.Printf("Web UI enabled: %t", cfg.EnableWebUI)
		log.Printf("Metrics enabled: %t", cfg.EnableMetrics)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	} else {
		log.Println("Server shutdown complete")
	}
}

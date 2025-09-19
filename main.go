package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/config"
	"github.com/johnwmail/nclip/handlers"
	"github.com/johnwmail/nclip/storage"

	// Lambda imports (only used when in Lambda mode)
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
)

// Lambda-specific variables
var (
	ginLambda     *ginadapter.GinLambda
	ginLambdaOnce sync.Once
)

// isLambdaEnvironment detects if running in AWS Lambda
func isLambdaEnvironment() bool {
	return os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != ""
}

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize storage backend based on deployment mode
	var store storage.PasteStore
	var err error

	if isLambdaEnvironment() {
		// Lambda mode: Always use DynamoDB
		store, err = storage.NewDynamoStore(cfg.DynamoTable)
		if err != nil {
			log.Fatalf("Failed to initialize DynamoDB storage for Lambda: %v", err)
		}
		log.Println("Lambda mode: Using DynamoDB storage")
	} else {
		// Container mode: Always use MongoDB
		store, err = storage.NewMongoStore(cfg.MongoURL, "nclip")
		if err != nil {
			log.Fatalf("Failed to initialize MongoDB storage: %v", err)
		}
		log.Println("Container mode: Using MongoDB storage")
	}

	// Setup router
	router := setupRouter(store, cfg)

	// Check if running in Lambda environment
	if isLambdaEnvironment() {
		log.Println("Starting in AWS Lambda mode")
		ginLambdaOnce.Do(func() {
			ginLambda = ginadapter.New(router)
		})
		lambda.Start(lambdaHandler)
		return
	}

	// Run in container/server mode
	log.Println("Starting in HTTP server mode")
	runHTTPServer(router, cfg, store)
}

// lambdaHandler handles Lambda requests
func lambdaHandler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	ginLambdaOnce.Do(func() {
		// Defensive: ginLambda should already be initialized, but ensure it's not nil
		if ginLambda == nil {
			log.Fatal("ginLambda is not initialized")
		}
	})
	return ginLambda.ProxyWithContext(ctx, req)
}

// setupRouter creates and configures the Gin router
func setupRouter(store storage.PasteStore, cfg *config.Config) *gin.Engine {
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

	return router
}

// runHTTPServer starts the HTTP server for container mode
func runHTTPServer(router *gin.Engine, cfg *config.Config, store storage.PasteStore) {
	// Ensure cleanup on exit
	defer func() {
		if err := store.Close(); err != nil {
			log.Printf("Error closing storage: %v", err)
		}
	}()

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting nclip server on port %d", cfg.Port)
		log.Printf("Storage backend: MongoDB (container mode)")
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

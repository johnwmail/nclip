package main

import (
	"context"
	"encoding/json"
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

// Version/build info (set via -ldflags at build time)
var (
	Version    = "dev"
	BuildTime  = "unknown"
	CommitHash = "none"
)

// Lambda-specific variables
var (
	ginLambdaV1   *ginadapter.GinLambda
	ginLambdaV2   *ginadapter.GinLambdaV2
	ginLambdaOnce sync.Once
)

// isLambdaEnvironment detects if running in AWS Lambda
func isLambdaEnvironment() bool {
	return os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != ""
}

func main() {

	// Print version/build info at startup
	log.Printf("NCLIP Version: %s", Version)
	log.Printf("Build Time:    %s", BuildTime)
	log.Printf("Commit Hash:   %s", CommitHash)

	// Aggressive logging: print all environment variables
	log.Printf("[DEBUG] ENVIRONMENT VARIABLES:")
	for _, e := range os.Environ() {
		log.Printf("[ENV] %s", e)
	}

	// Load configuration
	cfg := config.LoadConfig()
	cfg.Version = Version
	cfg.BuildTime = BuildTime
	cfg.CommitHash = CommitHash

	// Aggressive logging: print config
	log.Printf("[DEBUG] Loaded config: %+v", cfg)

	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Aggressive logging: print which storage backend is being used
	var backend string
	if os.Getenv("NCLIP_BACKEND") != "" {
		backend = os.Getenv("NCLIP_BACKEND")
	} else {
		backend = "auto (default)"
	}
	log.Printf("[DEBUG] Storage backend selection: %s", backend)

	// Initialize storage backend based on deployment mode
	var store storage.PasteStore
	var err error

	if isLambdaEnvironment() {
		// Lambda mode: Use S3
		store, err = storage.NewS3Store(cfg.S3Bucket)
		if err != nil {
			log.Fatalf("Failed to initialize S3 storage for Lambda: %v", err)
		}
		if os.Getenv("GIN_MODE") == "debug" {
			log.Printf("S3 Bucket: %s", cfg.S3Bucket)
		}
		log.Println("Lambda mode: Using S3 storage")
	} else {
		// Server mode: Use filesystem
		store, err = storage.NewFilesystemStore()
		if err != nil {
			log.Fatalf("Failed to initialize filesystem storage: %v", err)
		}
		log.Println("Server mode: Using filesystem storage")
		if os.Getenv("GIN_MODE") == "debug" {
			log.Printf("Listening on port: %d", cfg.Port)
		}
	}

	// Setup router
	router := setupRouter(store, cfg)

	// Check if running in Lambda environment
	if isLambdaEnvironment() {
		log.Println("Starting in AWS Lambda mode")
		ginLambdaOnce.Do(func() {
			ginLambdaV1 = ginadapter.New(router)
			ginLambdaV2 = ginadapter.NewV2(router)
		})
		lambda.Start(lambdaHandler)
		return
	}

	// Run in container/server mode
	log.Println("Starting in HTTP server mode")
	runHTTPServer(router, cfg, store)
}

// lambdaHandler handles Lambda requests for both v1 and v2 formats
func lambdaHandler(ctx context.Context, event interface{}) (interface{}, error) {
	ginLambdaOnce.Do(func() {
		// Defensive: adapters should already be initialized, but ensure they're not nil
		if ginLambdaV1 == nil || ginLambdaV2 == nil {
			log.Fatal("Lambda adapters are not initialized")
		}
	})

	// Log the raw event for debugging
	log.Printf("Received event type: %T", event)

	// Convert event to JSON bytes for parsing
	eventBytes, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       "Failed to process event",
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
		}, err
	}

	// Try to parse as APIGatewayV2HTTPRequest first (for Lambda Function URLs and HTTP API)
	var reqV2 events.APIGatewayV2HTTPRequest
	if err := json.Unmarshal(eventBytes, &reqV2); err == nil && reqV2.RequestContext.HTTP.Method != "" {
		log.Printf("Handling as APIGatewayV2HTTPRequest (Lambda Function URL/HTTP API)")
		log.Printf("Method: %s, Path: %s", reqV2.RequestContext.HTTP.Method, reqV2.RawPath)
		return ginLambdaV2.ProxyWithContext(ctx, reqV2)
	}

	// Try to parse as APIGatewayProxyRequest (for REST API and ALB)
	var reqV1 events.APIGatewayProxyRequest
	if err := json.Unmarshal(eventBytes, &reqV1); err == nil && reqV1.HTTPMethod != "" {
		log.Printf("Handling as APIGatewayProxyRequest (REST API/ALB)")
		log.Printf("Method: %s, Path: %s", reqV1.HTTPMethod, reqV1.Path)
		return ginLambdaV1.ProxyWithContext(ctx, reqV1)
	}

	// If neither format works, log the event structure and return error
	log.Printf("Unable to parse event as APIGateway v1 or v2 format")
	log.Printf("Event JSON: %s", string(eventBytes))

	// Check if this is a Lambda test event (contains test keys like key1, key2, key3)
	var testEvent map[string]interface{}
	if err := json.Unmarshal(eventBytes, &testEvent); err == nil {
		if _, hasKey1 := testEvent["key1"]; hasKey1 {
			log.Printf("Detected Lambda test event, returning success response")
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 200,
				Body:       `{"message": "nclip Lambda function is working! Use a real HTTP request or API Gateway integration."}`,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			}, nil
		}
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: 500,
		Body:       "Unsupported event type - this function expects API Gateway or Lambda Function URL events",
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
	}, fmt.Errorf("unsupported event type: %T", event)
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

	// Load favicon
	router.StaticFile("/favicon.ico", "./static/favicon.ico")

	// Load HTML templates
	router.LoadHTMLGlob("static/*.html")

	// Serve static files
	router.Static("/static", "./static")

	// Web UI routes (always enabled)
	router.GET("/", webuiHandler.Index)

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

	// Global 404 handler
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
	})

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
		log.Printf("Web UI: enabled")

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

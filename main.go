package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnwmail/nclip/config"
	"github.com/johnwmail/nclip/handlers"
	"github.com/johnwmail/nclip/handlers/retrieval"
	"github.com/johnwmail/nclip/handlers/upload"
	"github.com/johnwmail/nclip/internal/services"
	"github.com/johnwmail/nclip/storage"
	"github.com/johnwmail/nclip/utils"

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

	// Load configuration
	cfg := config.LoadConfig()
	cfg.Version = Version
	cfg.BuildTime = BuildTime
	cfg.CommitHash = CommitHash

	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Print the NCLIP_UPLOAD_AUTH settings at startup
	log.Printf("Upload Authentication Enabled: %v", cfg.UploadAuth)
	if cfg.UploadAuth {
		// Print the number of configured API keys without exposing them
		// Count and log the number of non-empty API keys (do not print the keys themselves)
		keys := strings.Split(cfg.APIKeys, ",")
		numKeys := 0
		for _, k := range keys {
			if strings.TrimSpace(k) != "" {
				numKeys++
			}
		}
		log.Printf("Configured API Keys: %d", numKeys)
	}

	// Aggressive logging: print all environment variables
	if utils.IsDebugEnabled() {
		log.Printf("[DEBUG] ENVIRONMENT VARIABLES:")
		for _, e := range os.Environ() {
			log.Printf("[ENV] %s", e)
		}
	}

	// Aggressive logging: print config
	if utils.IsDebugEnabled() {
		log.Printf("[DEBUG] Loaded config: %+v", cfg)
	}

	// Initialize storage backend based on deployment mode
	var store storage.PasteStore
	var err error

	if isLambdaEnvironment() {
		// Lambda mode: Use S3
		store, err = storage.NewS3Store(cfg.S3Bucket, cfg.S3Prefix)
		if err != nil {
			log.Fatalf("Failed to initialize S3 storage for Lambda: %v", err)
		}
		if utils.IsDebugEnabled() {
			log.Printf("S3 Bucket: %s", cfg.S3Bucket)
			log.Printf("S3 Prefix: %s", cfg.S3Prefix)
		}
		log.Println("Lambda mode: Using S3 storage")
	} else {
		// Server mode: Use filesystem. Use configured DataDir.
		store, err = storage.NewFilesystemStore(cfg.DataDir)
		if err != nil {
			log.Fatalf("Failed to initialize filesystem storage: %v", err)
		}
		log.Println("Server mode: Using filesystem storage")
		if utils.IsDebugEnabled() {
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
	// Initialize service
	pasteService := services.NewPasteService(store, cfg)

	// Initialize handlers
	uploadHandler := upload.NewHandler(pasteService, cfg)
	retrievalHandler := retrieval.NewHandler(pasteService, store, cfg)
	metaHandler := handlers.NewMetaHandler(store)
	systemHandler := handlers.NewSystemHandler()
	webuiHandler := handlers.NewWebUIHandler(cfg)

	// Create Gin router
	router := gin.New()

	// Add logging middleware
	// Use a JSON-safe recovery middleware and canonicalErrors middleware so
	// API endpoints always return JSON error responses instead of HTML error
	// pages that the web UI cannot parse.
	router.Use(gin.Logger())
	router.Use(jsonRecovery())
	router.Use(canonicalErrors())
	router.Use(gin.Recovery())

	// Load favicon
	router.StaticFile("/favicon.ico", "./static/favicon.ico")

	// Load HTML templates
	router.LoadHTMLGlob("static/*.html")

	// Serve static files
	router.Static("/static", "./static")

	// Web UI routes
	router.GET("/", webuiHandler.Index)

	// Core API routes
	if cfg.UploadAuth {
		auth := apiKeyAuth(cfg)
		router.POST("/", auth, uploadHandler.Upload)
		router.POST("/burn/", auth, uploadHandler.UploadBurn)
	} else {
		router.POST("/", uploadHandler.Upload)
		router.POST("/burn/", uploadHandler.UploadBurn)
	}
	router.GET("/:slug", retrievalHandler.View)
	router.GET("/raw/:slug", retrievalHandler.Raw)

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

// jsonRecovery returns a middleware that recovers from panics and ensures
// the response is JSON formatted so the web UI can parse error responses.
func jsonRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic for diagnostics
				log.Printf("[PANIC] %v", r)
				c.Header("Content-Type", "application/json; charset=utf-8")
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			}
		}()
		c.Next()
	}
}

// canonicalErrors ensures that if a handler did not write a body but the
// response status is an error (>=400), a small JSON error body is writtencanonicalErrors.
// This helps intermediaries and CDNs forward a predictable JSON payload.
func canonicalErrors() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Wrap the ResponseWriter so we can buffer the body and inspect it
		origWriter := c.Writer
		bcw := &bodyCaptureWriter{ResponseWriter: origWriter}
		c.Writer = bcw

		c.Next()

		status := bcw.Status()
		// Read buffered body and content-type
		buf := bcw.body.Bytes()
		ct := bcw.Header().Get("Content-Type")

		if status >= 400 {
			// Determine a suitable message to expose
			var msg string

			// If there is JSON body, try extracting its message/error
			if len(buf) > 0 && strings.Contains(ct, "application/json") {
				var parsed map[string]interface{}
				if err := json.Unmarshal(buf, &parsed); err == nil {
					if e, ok := parsed["error"].(string); ok {
						msg = e
					} else if m, ok := parsed["message"].(string); ok {
						msg = m
					}
				}
			}

			// If not found, use raw body text if present
			if msg == "" {
				if len(buf) > 0 {
					msg = string(bytes.TrimSpace(buf))
				} else if len(c.Errors) > 0 {
					msg = c.Errors.Last().Error()
				} else {
					msg = http.StatusText(status)
				}
			}

			// Write canonical JSON to the original writer
			origWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
			origWriter.WriteHeader(status)
			out, _ := json.Marshal(gin.H{"error": msg})
			if _, err := origWriter.Write(out); err != nil {
				log.Printf("[ERROR] canonicalErrors: failed to write error response: %v", err)
			}
			return
		}

		// Non-error: forward buffered content as-is
		if len(buf) > 0 {
			// Ensure headers/status are flushed
			origWriter.WriteHeader(status)
			if _, err := origWriter.Write(buf); err != nil {
				log.Printf("[ERROR] canonicalErrors: failed to write response body: %v", err)
			}
		}
	}
}

// apiKeyAuth returns a middleware that validates API keys supplied via
// Authorization: Bearer <key> or X-Api-Key: <key> headers. It reads keys
// from cfg.APIKeys (comma-separated) and denies unauthorized requests with
// HTTP 401.
func apiKeyAuth(cfg *config.Config) gin.HandlerFunc {
	// Build a map of allowed keys for fast lookup
	allowed := map[string]struct{}{}
	for _, k := range strings.Split(cfg.APIKeys, ",") {
		kk := strings.TrimSpace(k)
		if kk != "" {
			allowed[kk] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		// Extract key from Authorization: Bearer <key>
		var key string
		if auth := c.GetHeader("Authorization"); auth != "" {
			if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
				key = strings.TrimSpace(auth[7:])
			}
		}
		// If not found, try X-Api-Key
		if key == "" {
			key = strings.TrimSpace(c.GetHeader("X-Api-Key"))
		}

		if key == "" {
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing api key"})
			return
		}

		if _, ok := allowed[key]; !ok {
			// constant-time compare could be added, but we are checking map membership
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

// bodyCaptureWriter buffers response body writes so middleware can inspect
// and optionally rewrite the output before sending to the client.
type bodyCaptureWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

// Write implements io.Writer; buffer the bytes but do not write to the
// underlying writer until the middleware decides to forward them.
func (w *bodyCaptureWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
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

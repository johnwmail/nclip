package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/johnwmail/nclip/internal/config"
	"github.com/johnwmail/nclip/internal/server"
	"github.com/johnwmail/nclip/internal/storage"
)

var (
	httpServer *server.HTTPServer
	logger     *slog.Logger
)

func init() {
	// Set up logger
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Load configuration
	cfg := config.DefaultConfig()

	// Override with environment variables if present
	if storageType := os.Getenv("PASTEBIN_STORAGE_TYPE"); storageType != "" {
		cfg.StorageType = storageType
	}
	if domain := os.Getenv("PASTEBIN_DOMAIN"); domain != "" {
		cfg.BaseURL = domain
	}
	if url := os.Getenv("NCLIP_URL"); url != "" {
		cfg.BaseURL = url
	}
	if tableName := os.Getenv("PASTEBIN_DYNAMODB_TABLE"); tableName != "" {
		cfg.DynamoDBTable = tableName
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}

	// Create storage backend
	store, err := storage.NewStorage(cfg, logger)
	if err != nil {
		logger.Error("Failed to create storage backend", "error", err)
		os.Exit(1)
	}

	// Create HTTP server instance
	httpServer = server.NewHTTPServer(cfg, store, logger)
	if httpServer == nil {
		logger.Error("Failed to create HTTP server")
		os.Exit(1)
	}

	logger.Info("Lambda function initialized",
		"storage_type", cfg.StorageType,
		"base_url", cfg.BaseURL,
		"expire_days", cfg.ExpireDays)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger.Debug("Processing request",
		"method", request.HTTPMethod,
		"path", request.Path,
		"headers", request.Headers)

	// Convert API Gateway request to HTTP request
	httpReq, err := convertAPIGatewayRequestToHTTP(request)
	if err != nil {
		logger.Error("Failed to convert API Gateway request", "error", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Internal server error",
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
		}, nil
	}

	// Create response recorder
	recorder := &ResponseRecorder{
		statusCode: 200,
		headers:    make(http.Header),
		body:       strings.Builder{},
	}

	// Handle the request using the HTTP handler
	handler := httpServer.GetHandler()
	handler.ServeHTTP(recorder, httpReq)

	// Convert response
	response := events.APIGatewayProxyResponse{
		StatusCode: recorder.statusCode,
		Headers:    make(map[string]string),
		Body:       recorder.body.String(),
	}

	// Copy headers
	for key, values := range recorder.headers {
		if len(values) > 0 {
			response.Headers[key] = values[0]
		}
	}

	logger.Debug("Response prepared",
		"status_code", response.StatusCode,
		"body_length", len(response.Body))

	return response, nil
}

func convertAPIGatewayRequestToHTTP(request events.APIGatewayProxyRequest) (*http.Request, error) {
	// Construct URL
	scheme := "https"
	if request.Headers["X-Forwarded-Proto"] != "" {
		scheme = request.Headers["X-Forwarded-Proto"]
	}

	host := request.Headers["Host"]
	if host == "" {
		host = "localhost"
	}

	url := fmt.Sprintf("%s://%s%s", scheme, host, request.Path)
	if len(request.QueryStringParameters) > 0 {
		params := make([]string, 0, len(request.QueryStringParameters))
		for key, value := range request.QueryStringParameters {
			params = append(params, fmt.Sprintf("%s=%s", key, value))
		}
		url += "?" + strings.Join(params, "&")
	}

	// Handle request body
	var body string
	if request.Body != "" {
		if request.IsBase64Encoded {
			decoded, err := base64.StdEncoding.DecodeString(request.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to decode base64 body: %w", err)
			}
			body = string(decoded)
		} else {
			body = request.Body
		}
	}

	// Create HTTP request
	httpReq, err := http.NewRequest(request.HTTPMethod, url, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Copy headers
	for key, value := range request.Headers {
		httpReq.Header.Set(key, value)
	}

	// Copy multi-value headers
	for key, values := range request.MultiValueHeaders {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

	// Set remote address for rate limiting
	if clientIP := request.Headers["X-Forwarded-For"]; clientIP != "" {
		// X-Forwarded-For may contain multiple IPs, take the first one
		if ips := strings.Split(clientIP, ","); len(ips) > 0 {
			httpReq.RemoteAddr = strings.TrimSpace(ips[0]) + ":0"
		}
	} else if realIP := request.Headers["X-Real-IP"]; realIP != "" {
		httpReq.RemoteAddr = realIP + ":0"
	} else {
		httpReq.RemoteAddr = request.RequestContext.Identity.SourceIP + ":0"
	}

	return httpReq, nil
}

// ResponseRecorder implements http.ResponseWriter to capture the response
type ResponseRecorder struct {
	statusCode int
	headers    http.Header
	body       strings.Builder
}

func (r *ResponseRecorder) Header() http.Header {
	return r.headers
}

func (r *ResponseRecorder) Write(data []byte) (int, error) {
	return r.body.Write(data)
}

func (r *ResponseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

func main() {
	lambda.Start(handler)
}

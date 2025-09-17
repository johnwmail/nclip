package dynamodb

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"log/slog"

	"github.com/aws/aws-lambda-go/events"
	"github.com/johnwmail/nclip/internal/slug"
	"github.com/johnwmail/nclip/internal/storage"
)

// Use shared Paste struct from storage

var (
	dynamoStorage storage.Storage
	slugGenerator *slug.Generator
)

func InitDynamoStorage() {
	tableName := os.Getenv("NCLIP_DYNAMODB_TABLE")
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	s, err := storage.NewDynamoDBStorage(tableName, logger)
	if err != nil {
		panic(fmt.Sprintf("unable to initialize DynamoDB storage: %v", err))
	}
	dynamoStorage = s

	// Initialize slug generator
	slugGenerator = slug.New(8) // Use 8-character slugs like default config
}

func Handler(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	switch req.RequestContext.HTTP.Method {
	case "POST":
		return createPaste(ctx, req)
	case "GET":
		return getPaste(ctx, req)
	default:
		return events.LambdaFunctionURLResponse{StatusCode: 405}, nil
	}
}

func createPaste(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	// Read content from request body (similar to HTTP server)
	content := []byte(req.Body)

	if len(content) == 0 {
		return events.LambdaFunctionURLResponse{
			StatusCode: 400,
			Body:       "Empty paste",
			Headers:    map[string]string{"Content-Type": "text/plain"},
		}, nil
	}

	// Ensure content ends with newline (like original fiche behavior)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		content = append(content, '\n')
	}

	// Generate unique slug
	slugStr, err := slugGenerator.GenerateWithCollisionCheck(dynamoStorage.Exists)
	if err != nil {
		return events.LambdaFunctionURLResponse{
			StatusCode: 500,
			Body:       "Could not generate paste ID",
			Headers:    map[string]string{"Content-Type": "text/plain"},
		}, nil
	}

	// Extract metadata from headers (similar to HTTP server)
	contentType := "text/plain"
	if ct := req.Headers["content-type"]; ct != "" {
		contentType = ct
	}

	filename := req.Headers["x-filename"]
	language := req.Headers["x-language"]
	title := req.Headers["x-title"]

	// Create paste
	paste := &storage.Paste{
		ID:          slugStr,
		Content:     content,
		ContentType: contentType,
		Filename:    filename,
		Language:    language,
		Title:       title,
		CreatedAt:   time.Now(),
		ClientIP:    req.RequestContext.HTTP.SourceIP,
		Size:        int64(len(content)),
	}

	// Set default expiration (30 days like in the original code)
	expires := paste.CreatedAt.Add(30 * 24 * time.Hour)
	paste.ExpiresAt = &expires

	if err := dynamoStorage.Store(paste); err != nil {
		return events.LambdaFunctionURLResponse{
			StatusCode: 500,
			Body:       err.Error(),
			Headers:    map[string]string{"Content-Type": "text/plain"},
		}, nil
	}

	// Return the paste ID in the body (like the HTTP server returns URL)
	respBody, _ := json.Marshal(map[string]string{"id": paste.ID})
	return events.LambdaFunctionURLResponse{
		StatusCode: 201,
		Body:       string(respBody),
		Headers:    map[string]string{"Content-Type": "application/json"},
	}, nil
}

func getPaste(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	// Extract ID from the URL path - for Lambda Function URLs, we need to parse the raw path
	path := req.RequestContext.HTTP.Path
	// Assuming the path is like /paste/{id} or just /{id}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	var id string
	if len(parts) > 0 && parts[len(parts)-1] != "" {
		id = parts[len(parts)-1]
	}

	if id == "" {
		return events.LambdaFunctionURLResponse{
			StatusCode: 400,
			Body:       "Missing paste ID",
			Headers:    map[string]string{"Content-Type": "text/plain"},
		}, nil
	}

	paste, err := dynamoStorage.Get(id)
	if err != nil {
		return events.LambdaFunctionURLResponse{
			StatusCode: 404,
			Body:       "Paste not found",
			Headers:    map[string]string{"Content-Type": "text/plain"},
		}, nil
	}

	// Check expiration
	if paste.ExpiresAt != nil && time.Now().After(*paste.ExpiresAt) {
		return events.LambdaFunctionURLResponse{
			StatusCode: 410,
			Body:       "Paste has expired",
			Headers:    map[string]string{"Content-Type": "text/plain"},
		}, nil
	}

	respBody, _ := json.Marshal(paste)
	return events.LambdaFunctionURLResponse{
		StatusCode: 200,
		Body:       string(respBody),
		Headers:    map[string]string{"Content-Type": "application/json"},
	}, nil
}

// ...existing code...

package dynamodb

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"log/slog"

	"github.com/aws/aws-lambda-go/events"
	"github.com/johnwmail/nclip/internal/storage"
)

// Use shared Paste struct from storage

var (
	dynamoStorage storage.Storage
)

func InitDynamoStorage() {
	tableName := os.Getenv("NCLIP_DYNAMODB_TABLE")
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	s, err := storage.NewDynamoDBStorage(tableName, logger)
	if err != nil {
		panic(fmt.Sprintf("unable to initialize DynamoDB storage: %v", err))
	}
	dynamoStorage = s
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	switch req.HTTPMethod {
	case "POST":
		return createPaste(ctx, req)
	case "GET":
		return getPaste(ctx, req)
	default:
		return events.APIGatewayProxyResponse{StatusCode: 405}, nil
	}
}

func createPaste(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var paste storage.Paste
	if err := json.Unmarshal([]byte(req.Body), &paste); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Invalid request",
			Headers:    map[string]string{"Content-Type": "text/plain"},
		}, nil
	}

	// Validate required fields
	if paste.ID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Missing paste ID",
			Headers:    map[string]string{"Content-Type": "text/plain"},
		}, nil
	}

	paste.CreatedAt = time.Now()
	if paste.ExpiresAt == nil {
		expires := paste.CreatedAt.Add(30 * 24 * time.Hour)
		paste.ExpiresAt = &expires
	}

	if err := dynamoStorage.Store(&paste); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       err.Error(),
			Headers:    map[string]string{"Content-Type": "text/plain"},
		}, nil
	}
	respBody, _ := json.Marshal(map[string]string{"id": paste.ID})
	return events.APIGatewayProxyResponse{
		StatusCode: 201,
		Body:       string(respBody),
		Headers:    map[string]string{"Content-Type": "application/json"},
	}, nil
}

func getPaste(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id := req.PathParameters["id"]
	if id == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Missing paste ID",
			Headers:    map[string]string{"Content-Type": "text/plain"},
		}, nil
	}

	paste, err := dynamoStorage.Get(id)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 404,
			Body:       "Paste not found",
			Headers:    map[string]string{"Content-Type": "text/plain"},
		}, nil
	}

	// Check expiration
	if paste.ExpiresAt != nil && time.Now().After(*paste.ExpiresAt) {
		return events.APIGatewayProxyResponse{
			StatusCode: 410,
			Body:       "Paste has expired",
			Headers:    map[string]string{"Content-Type": "text/plain"},
		}, nil
	}

	respBody, _ := json.Marshal(paste)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(respBody),
		Headers:    map[string]string{"Content-Type": "application/json"},
	}, nil
}

// ...existing code...

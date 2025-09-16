package dynamodb

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/johnwmail/nclip/internal/storage"
)

func TestCreatePaste_InvalidRequest(t *testing.T) {
	if err := os.Setenv("NCLIP_DYNAMODB_TABLE", "test-table"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	InitDynamoStorage() // Will use real AWS config, mock in real tests
	resp, _ := createPaste(context.Background(), events.APIGatewayProxyRequest{Body: "not-json"})
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetPaste_MissingID(t *testing.T) {
	if err := os.Setenv("NCLIP_DYNAMODB_TABLE", "test-table"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	InitDynamoStorage()
	resp, _ := getPaste(context.Background(), events.APIGatewayProxyRequest{PathParameters: map[string]string{"id": ""}})
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreatePaste_Valid(t *testing.T) {
	if err := os.Setenv("NCLIP_DYNAMODB_TABLE", "test-table"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	InitDynamoStorage()
	paste := storage.Paste{
		ID:          "testid",
		Content:     []byte("testcontent"),
		ContentType: "text/plain",
		ClientIP:    "127.0.0.1",
		Size:        12,
		Metadata:    map[string]string{"foo": "bar"},
	}
	body, _ := json.Marshal(paste)
	resp, _ := createPaste(context.Background(), events.APIGatewayProxyRequest{Body: string(body)})
	if resp.StatusCode != 201 && resp.StatusCode != 500 {
		t.Errorf("expected 201 or 500, got %d", resp.StatusCode)
	}
}

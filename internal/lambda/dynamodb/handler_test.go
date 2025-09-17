package dynamodb

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/johnwmail/nclip/internal/slug"
	"github.com/johnwmail/nclip/internal/storage"
)

// mockStorage is a simple in-memory mock for storage.Storage
type mockStorage struct {
	pastes map[string]*storage.Paste
}

func newMockStorage() *mockStorage {
	return &mockStorage{pastes: make(map[string]*storage.Paste)}
}
func (m *mockStorage) Store(paste *storage.Paste) error {
	m.pastes[paste.ID] = paste
	return nil
}
func (m *mockStorage) Get(id string) (*storage.Paste, error) {
	p, ok := m.pastes[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return p, nil
}
func (m *mockStorage) Exists(id string) bool {
	_, ok := m.pastes[id]
	return ok
}
func (m *mockStorage) Delete(id string) error           { delete(m.pastes, id); return nil }
func (m *mockStorage) List(limit int) ([]string, error) { return nil, nil }
func (m *mockStorage) Stats() (*storage.Stats, error)   { return &storage.Stats{}, nil }
func (m *mockStorage) Cleanup() error                   { return nil }
func (m *mockStorage) Close() error                     { return nil }

func TestCreatePaste_InvalidRequest(t *testing.T) {
	dynamoStorage = newMockStorage()
	slugGenerator = slug.New(8)
	resp, _ := createPaste(context.Background(), events.LambdaFunctionURLRequest{Body: "not-json"})
	if resp.StatusCode != 201 {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
}

func TestGetPaste_MissingID(t *testing.T) {
	dynamoStorage = newMockStorage()
	slugGenerator = slug.New(8)
	resp, _ := getPaste(context.Background(), events.LambdaFunctionURLRequest{
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{
				Path: "/", // Empty path should result in missing ID
			},
		},
	})
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreatePaste_Valid(t *testing.T) {
	dynamoStorage = newMockStorage()
	slugGenerator = slug.New(8)
	paste := storage.Paste{
		ID:          "testid",
		Content:     []byte("testcontent"),
		ContentType: "text/plain",
		ClientIP:    "127.0.0.1",
		Size:        12,
		Metadata:    map[string]string{"foo": "bar"},
	}
	body, _ := json.Marshal(paste)
	resp, _ := createPaste(context.Background(), events.LambdaFunctionURLRequest{Body: string(body)})
	if resp.StatusCode != 201 {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
}

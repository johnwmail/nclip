package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoDBStorage implements Storage interface using AWS DynamoDB
type DynamoDBStorage struct {
	client    *dynamodb.Client
	tableName string
	logger    *slog.Logger
}

// DynamoDBPaste represents a paste item in DynamoDB
type DynamoDBPaste struct {
	ID          string    `dynamodbav:"id" json:"id"`
	Content     []byte    `dynamodbav:"content" json:"content"`
	ContentType string    `dynamodbav:"content_type" json:"content_type"`
	CreatedAt   time.Time `dynamodbav:"created_at" json:"created_at"`
	ExpiresAt   int64     `dynamodbav:"expires_at" json:"expires_at"` // Unix timestamp for TTL
	ClientIP    string    `dynamodbav:"client_ip" json:"client_ip"`
	Filename    string    `dynamodbav:"filename,omitempty" json:"filename,omitempty"`
	Language    string    `dynamodbav:"language,omitempty" json:"language,omitempty"`
	Title       string    `dynamodbav:"title,omitempty" json:"title,omitempty"`
	Size        int64     `dynamodbav:"size" json:"size"`
}

// NewDynamoDBStorage creates a new DynamoDB storage instance
func NewDynamoDBStorage(tableName string, logger *slog.Logger) (*DynamoDBStorage, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS SDK config: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	return &DynamoDBStorage{
		client:    client,
		tableName: tableName,
		logger:    logger,
	}, nil
}

// Store saves a paste to DynamoDB
func (d *DynamoDBStorage) Store(paste *Paste) error {
	if d.client == nil {
		return fmt.Errorf("DynamoDB client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Convert paste to DynamoDB item
	item := map[string]types.AttributeValue{
		"id":           &types.AttributeValueMemberS{Value: paste.ID},
		"content":      &types.AttributeValueMemberB{Value: paste.Content},
		"content_type": &types.AttributeValueMemberS{Value: paste.ContentType},
		"created_at":   &types.AttributeValueMemberN{Value: strconv.FormatInt(paste.CreatedAt.Unix(), 10)},
		"client_ip":    &types.AttributeValueMemberS{Value: paste.ClientIP},
		"size":         &types.AttributeValueMemberN{Value: strconv.FormatInt(paste.Size, 10)},
	}

	// Add optional fields
	if paste.Filename != "" {
		item["filename"] = &types.AttributeValueMemberS{Value: paste.Filename}
	}
	if paste.Language != "" {
		item["language"] = &types.AttributeValueMemberS{Value: paste.Language}
	}
	if paste.Title != "" {
		item["title"] = &types.AttributeValueMemberS{Value: paste.Title}
	}
	if paste.ExpiresAt != nil {
		item["expires_at"] = &types.AttributeValueMemberN{Value: strconv.FormatInt(paste.ExpiresAt.Unix(), 10)}
	}

	// Add metadata if present
	if len(paste.Metadata) > 0 {
		metaBytes, err := json.Marshal(paste.Metadata)
		if err != nil {
			d.logger.Error("Failed to marshal metadata", "error", err)
		} else {
			item["metadata"] = &types.AttributeValueMemberS{Value: string(metaBytes)}
		}
	}

	_, err := d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item:      item,
	})

	if err != nil {
		d.logger.Error("Failed to store paste in DynamoDB", "id", paste.ID, "error", err)
		return fmt.Errorf("failed to store paste: %w", err)
	}

	d.logger.Debug("Paste stored successfully in DynamoDB", "id", paste.ID, "size", paste.Size)
	return nil
}

// Get retrieves a paste from DynamoDB
func (d *DynamoDBStorage) Get(id string) (*Paste, error) {
	return nil, fmt.Errorf("DynamoDB storage not implemented - add AWS SDK dependencies")
}

// Exists checks if a paste exists in DynamoDB
func (d *DynamoDBStorage) Exists(id string) bool {
	return false
}

// Delete removes a paste by ID
func (d *DynamoDBStorage) Delete(id string) error {
	return fmt.Errorf("DynamoDB storage not implemented - add AWS SDK dependencies")
}

// List returns a list of paste IDs
func (d *DynamoDBStorage) List(limit int) ([]string, error) {
	return nil, fmt.Errorf("DynamoDB storage not implemented - add AWS SDK dependencies")
}

// Stats returns storage statistics
func (d *DynamoDBStorage) Stats() (*Stats, error) {
	return &Stats{
		TotalPastes:   0,
		TotalSize:     0,
		ExpiredPastes: 0,
	}, nil
}

// Cleanup is not needed for DynamoDB as TTL handles expiration automatically
func (d *DynamoDBStorage) Cleanup() error {
	// DynamoDB TTL automatically deletes expired items
	d.logger.Info("DynamoDB TTL handles automatic cleanup")
	return nil
}

// Close closes the DynamoDB connection
func (d *DynamoDBStorage) Close() error {
	d.logger.Info("DynamoDB connection closed")
	return nil
}

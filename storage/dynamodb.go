package storage

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/johnwmail/nclip/models"
)

// DynamoStore implements PasteStore using DynamoDB
type DynamoStore struct {
	client    *dynamodb.Client
	tableName string
}

// NewDynamoStore creates a new DynamoDB storage backend
func NewDynamoStore(tableName, region string) (*DynamoStore, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}

	client := dynamodb.NewFromConfig(cfg)

	return &DynamoStore{
		client:    client,
		tableName: tableName,
	}, nil
}

// Store saves a paste to DynamoDB
func (d *DynamoStore) Store(paste *models.Paste) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	item := map[string]types.AttributeValue{
		"id":              &types.AttributeValueMemberS{Value: paste.ID},
		"created_at":      &types.AttributeValueMemberN{Value: strconv.FormatInt(paste.CreatedAt.Unix(), 10)},
		"size":            &types.AttributeValueMemberN{Value: strconv.FormatInt(paste.Size, 10)},
		"content_type":    &types.AttributeValueMemberS{Value: paste.ContentType},
		"burn_after_read": &types.AttributeValueMemberBOOL{Value: paste.BurnAfterRead},
		"read_count":      &types.AttributeValueMemberN{Value: strconv.Itoa(paste.ReadCount)},
		"content":         &types.AttributeValueMemberB{Value: paste.Content},
	}

	// Add TTL if expires_at is set
	if paste.ExpiresAt != nil {
		item["expires_at"] = &types.AttributeValueMemberN{Value: strconv.FormatInt(paste.ExpiresAt.Unix(), 10)}
		item["ttl"] = &types.AttributeValueMemberN{Value: strconv.FormatInt(paste.ExpiresAt.Unix(), 10)}
	}

	_, err := d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item:      item,
	})

	return err
}

// Get retrieves a paste by its ID
func (d *DynamoStore) Get(id string) (*models.Paste, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, nil // Not found
	}

	paste, err := d.itemToPaste(result.Item)
	if err != nil {
		return nil, err
	}

	// Check if expired
	if paste.IsExpired() {
		// Delete expired paste
		d.Delete(id)
		return nil, nil
	}

	return paste, nil
}

// Delete removes a paste from DynamoDB
func (d *DynamoStore) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := d.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})

	return err
}

// IncrementReadCount increments the read count for a paste
func (d *DynamoStore) IncrementReadCount(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression: aws.String("ADD read_count :inc"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":inc": &types.AttributeValueMemberN{Value: "1"},
		},
	})

	return err
}

// Close is a no-op for DynamoDB
func (d *DynamoStore) Close() error {
	return nil
}

// itemToPaste converts a DynamoDB item to a Paste model
func (d *DynamoStore) itemToPaste(item map[string]types.AttributeValue) (*models.Paste, error) {
	paste := &models.Paste{}

	if id, ok := item["id"].(*types.AttributeValueMemberS); ok {
		paste.ID = id.Value
	}

	if createdAt, ok := item["created_at"].(*types.AttributeValueMemberN); ok {
		if timestamp, err := strconv.ParseInt(createdAt.Value, 10, 64); err == nil {
			paste.CreatedAt = time.Unix(timestamp, 0)
		}
	}

	if expiresAt, ok := item["expires_at"].(*types.AttributeValueMemberN); ok {
		if timestamp, err := strconv.ParseInt(expiresAt.Value, 10, 64); err == nil {
			expiry := time.Unix(timestamp, 0)
			paste.ExpiresAt = &expiry
		}
	}

	if size, ok := item["size"].(*types.AttributeValueMemberN); ok {
		if s, err := strconv.ParseInt(size.Value, 10, 64); err == nil {
			paste.Size = s
		}
	}

	if contentType, ok := item["content_type"].(*types.AttributeValueMemberS); ok {
		paste.ContentType = contentType.Value
	}

	if burnAfterRead, ok := item["burn_after_read"].(*types.AttributeValueMemberBOOL); ok {
		paste.BurnAfterRead = burnAfterRead.Value
	}

	if readCount, ok := item["read_count"].(*types.AttributeValueMemberN); ok {
		if count, err := strconv.Atoi(readCount.Value); err == nil {
			paste.ReadCount = count
		}
	}

	if content, ok := item["content"].(*types.AttributeValueMemberB); ok {
		paste.Content = content.Value
	}

	return paste, nil
}

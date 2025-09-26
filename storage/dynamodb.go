package storage

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/johnwmail/nclip/models"
)

// Constants for chunking
const (
	MaxDynamoItemSize = 400 * 1024 // 400KB
	ChunkSize         = 390 * 1024 // 390KB per chunk to allow for metadata
)

// DynamoStore implements PasteStore using DynamoDB
type DynamoStore struct {
	client    *dynamodb.Client
	tableName string
}

// NewDynamoStore creates a new DynamoDB storage backend
// Uses the default AWS region configuration (from environment, IAM role, etc.)
func NewDynamoStore(tableName string) (*DynamoStore, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
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
	fmt.Printf("[DEBUG] DynamoStore.Store: id=%s, size=%d, is_chunked=%v\n", paste.ID, len(paste.Content), len(paste.Content) > ChunkSize)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	content := paste.Content
	if len(content) <= ChunkSize {
		// Store as single item (not chunked, but still use chunk_index = -1 for metadata)
		item := map[string]types.AttributeValue{
			"id":              &types.AttributeValueMemberS{Value: paste.ID},
			"created_at":      &types.AttributeValueMemberN{Value: strconv.FormatInt(paste.CreatedAt.Unix(), 10)},
			"size":            &types.AttributeValueMemberN{Value: strconv.FormatInt(paste.Size, 10)},
			"content_type":    &types.AttributeValueMemberS{Value: paste.ContentType},
			"burn_after_read": &types.AttributeValueMemberBOOL{Value: paste.BurnAfterRead},
			"read_count":      &types.AttributeValueMemberN{Value: strconv.Itoa(paste.ReadCount)},
			"content":         &types.AttributeValueMemberB{Value: content},
			"is_chunked":      &types.AttributeValueMemberBOOL{Value: false},
			"chunk_count":     &types.AttributeValueMemberN{Value: "1"},
			"chunk_index":     &types.AttributeValueMemberN{Value: "-1"},
		}
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

	// Chunked storage
	chunkCount := (len(content) + ChunkSize - 1) / ChunkSize
	paste.ChunkCount = chunkCount
	paste.IsChunked = true
	fmt.Printf("[DEBUG] Chunking: total size=%d, chunk size=%d, chunk count=%d\n", len(content), ChunkSize, chunkCount)

	// Store metadata item (no content)
	meta := map[string]types.AttributeValue{
		"id":              &types.AttributeValueMemberS{Value: paste.ID},
		"created_at":      &types.AttributeValueMemberN{Value: strconv.FormatInt(paste.CreatedAt.Unix(), 10)},
		"size":            &types.AttributeValueMemberN{Value: strconv.FormatInt(paste.Size, 10)},
		"content_type":    &types.AttributeValueMemberS{Value: paste.ContentType},
		"burn_after_read": &types.AttributeValueMemberBOOL{Value: paste.BurnAfterRead},
		"read_count":      &types.AttributeValueMemberN{Value: strconv.Itoa(paste.ReadCount)},
		"is_chunked":      &types.AttributeValueMemberBOOL{Value: true},
		"chunk_count":     &types.AttributeValueMemberN{Value: strconv.Itoa(chunkCount)},
		"chunk_index":     &types.AttributeValueMemberN{Value: "-1"},
	}
	if paste.ExpiresAt != nil {
		meta["expires_at"] = &types.AttributeValueMemberN{Value: strconv.FormatInt(paste.ExpiresAt.Unix(), 10)}
		meta["ttl"] = &types.AttributeValueMemberN{Value: strconv.FormatInt(paste.ExpiresAt.Unix(), 10)}
	}
	_, err := d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item:      meta,
	})
	if err != nil {
		return err
	}

	// Store each chunk as a separate item
	for i := 0; i < chunkCount; i++ {
		fmt.Printf("[DEBUG] Storing chunk %d/%d: bytes %d-%d\n", i+1, chunkCount, i*ChunkSize, min((i+1)*ChunkSize, len(content)))
		start := i * ChunkSize
		end := start + ChunkSize
		if end > len(content) {
			end = len(content)
		}
		chunk := content[start:end]
		chunkItem := map[string]types.AttributeValue{
			"id":          &types.AttributeValueMemberS{Value: paste.ID},
			"chunk_index": &types.AttributeValueMemberN{Value: strconv.Itoa(i)},
			"content":     &types.AttributeValueMemberB{Value: chunk},
		}
		if paste.ExpiresAt != nil {
			chunkItem["ttl"] = &types.AttributeValueMemberN{Value: strconv.FormatInt(paste.ExpiresAt.Unix(), 10)}
		}
		_, err := d.client.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(d.tableName),
			Item:      chunkItem,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Get retrieves a paste by its ID
func (d *DynamoStore) Get(id string) (*models.Paste, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get metadata item (chunk_index = -1)
	metaKey := map[string]types.AttributeValue{
		"id":          &types.AttributeValueMemberS{Value: id},
		"chunk_index": &types.AttributeValueMemberN{Value: "-1"},
	}
	result, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.tableName),
		Key:       metaKey,
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
		_ = d.Delete(id)
		return nil, nil
	}

	// If not chunked, return as usual
	if !paste.IsChunked {
		return paste, nil
	}

	// Chunked: fetch all chunk items and reassemble
	chunkCount := paste.ChunkCount
	var content []byte
	for i := 0; i < chunkCount; i++ {
		chunkKey := map[string]types.AttributeValue{
			"id":          &types.AttributeValueMemberS{Value: id},
			"chunk_index": &types.AttributeValueMemberN{Value: strconv.Itoa(i)},
		}
		chunkRes, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
			TableName: aws.String(d.tableName),
			Key:       chunkKey,
		})
		if err != nil {
			fmt.Printf("[ERROR] Failed to get chunk %d for paste %s: %v\n", i, id, err)
			return nil, err
		}
		if chunkRes.Item == nil {
			fmt.Printf("[ERROR] Missing chunk %d for paste %s\n", i, id)
			return nil, nil // Missing chunk
		}
		if chunkVal, ok := chunkRes.Item["content"].(*types.AttributeValueMemberB); ok {
			content = append(content, chunkVal.Value...)
		} else {
			fmt.Printf("[ERROR] Malformed chunk %d for paste %s\n", i, id)
			return nil, nil // Malformed chunk
		}
	}
	paste.Content = content
	return paste, nil
}

// Delete removes a paste from DynamoDB
func (d *DynamoStore) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get metadata to check if chunked
	result, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return err
	}
	if result.Item == nil {
		return nil // Not found
	}
	paste, err := d.itemToPaste(result.Item)
	if err != nil {
		return err
	}

	// Delete metadata item
	_, err = d.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return err
	}

	// If chunked, delete all chunk items
	if paste.IsChunked {
		for i := 0; i < paste.ChunkCount; i++ {
			chunkKey := map[string]types.AttributeValue{
				"id":          &types.AttributeValueMemberS{Value: id},
				"chunk_index": &types.AttributeValueMemberN{Value: strconv.Itoa(i)},
			}
			_, _ = d.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
				TableName: aws.String(d.tableName),
				Key:       chunkKey,
			})
		}
	}
	return nil
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
	if isChunked, ok := item["is_chunked"].(*types.AttributeValueMemberBOOL); ok {
		paste.IsChunked = isChunked.Value
	}
	if chunkCount, ok := item["chunk_count"].(*types.AttributeValueMemberN); ok {
		if cc, err := strconv.Atoi(chunkCount.Value); err == nil {
			paste.ChunkCount = cc
		}
	}
	return paste, nil
}

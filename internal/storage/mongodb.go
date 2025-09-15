package storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBStorage implements Storage interface using MongoDB
type MongoDBStorage struct {
	client     *mongo.Client
	database   string
	collection string
	logger     *slog.Logger
}

// MongoPaste represents a paste document in MongoDB
type MongoPaste struct {
	ID          string            `bson:"_id" json:"id"`
	Content     []byte            `bson:"content" json:"content"`
	ContentType string            `bson:"content_type" json:"content_type"`
	Filename    string            `bson:"filename,omitempty" json:"filename,omitempty"`
	Language    string            `bson:"language,omitempty" json:"language,omitempty"`
	Title       string            `bson:"title,omitempty" json:"title,omitempty"`
	CreatedAt   time.Time         `bson:"created_at" json:"created_at"`
	ExpiresAt   time.Time         `bson:"expires_at" json:"expires_at"`
	ClientIP    string            `bson:"client_ip" json:"client_ip"`
	Size        int64             `bson:"size" json:"size"`
	Metadata    map[string]string `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

// NewMongoDBStorage creates a new MongoDB storage instance
func NewMongoDBStorage(connectionURI, database, collection string, logger *slog.Logger) *MongoDBStorage {
	storage := &MongoDBStorage{
		database:   database,
		collection: collection,
		logger:     logger,
	}

	// Connect to MongoDB with a shorter timeout for testing
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionURI))
	if err != nil {
		logger.Error("Failed to connect to MongoDB", "error", err, "uri", connectionURI)
		logger.Warn("MongoDB storage will be unavailable - pastes cannot be saved")
		return storage // Return storage with nil client to indicate error
	}

	// Test the connection with a shorter timeout
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer pingCancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		logger.Error("Failed to ping MongoDB", "error", err)
		logger.Warn("MongoDB storage will be unavailable - pastes cannot be saved")
		// Close the connection since ping failed
		if err := client.Disconnect(context.Background()); err != nil {
			logger.Warn("Failed to disconnect MongoDB client after ping failure", "error", err)
		}
		return storage // Return storage with nil client to indicate error
	}

	storage.client = client
	logger.Info("Connected to MongoDB successfully",
		"database", database,
		"collection", collection)

	// Create TTL index for automatic expiration if it doesn't exist
	storage.ensureIndexes()

	return storage
}

// ensureIndexes creates necessary indexes for the collection
func (m *MongoDBStorage) ensureIndexes() {
	if m.client == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := m.client.Database(m.database).Collection(m.collection)

	// TTL index for automatic expiration
	ttlIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	}

	// Additional performance indexes
	createdAtIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "created_at", Value: 1}},
	}

	contentTypeIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "content_type", Value: 1}},
	}

	sizeIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "size", Value: 1}},
	}

	indexes := []mongo.IndexModel{ttlIndex, createdAtIndex, contentTypeIndex, sizeIndex}

	_, err := coll.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		m.logger.Warn("Failed to create indexes", "error", err)
	} else {
		m.logger.Debug("MongoDB indexes ensured")
	}
}

// Store saves a paste to MongoDB
func (m *MongoDBStorage) Store(paste *Paste) error {
	if m.client == nil {
		m.logger.Error("Cannot save paste - MongoDB connection not available",
			"id", paste.ID,
			"reason", "MongoDB connection failed during initialization")
		return fmt.Errorf("can't save paste: MongoDB connection not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	coll := m.client.Database(m.database).Collection(m.collection)

	// Convert to MongoDB document
	doc := bson.M{
		"_id":          paste.ID,
		"content":      primitive.Binary{Subtype: 0x00, Data: paste.Content}, // Store as binData
		"content_type": paste.ContentType,
		"filename":     paste.Filename,
		"language":     paste.Language,
		"title":        paste.Title,
		"created_at":   paste.CreatedAt,
		"client_ip":    paste.ClientIP,
		"size":         paste.Size,
	}

	// Ensure metadata is always an object (never null)
	if paste.Metadata == nil {
		doc["metadata"] = bson.M{}
	} else {
		metaObj := bson.M{}
		for k, v := range paste.Metadata {
			metaObj[k] = v
		}
		doc["metadata"] = metaObj
	}

	// Only add expires_at if it's not nil
	if paste.ExpiresAt != nil {
		doc["expires_at"] = *paste.ExpiresAt
	}

	_, err := coll.InsertOne(ctx, doc)
	if err != nil {
		m.logger.Error("Failed to store paste in MongoDB",
			"id", paste.ID,
			"error", err,
			"database", m.database,
			"collection", m.collection)
		return fmt.Errorf("can't save paste: %w", err)
	}

	m.logger.Debug("Paste stored successfully in MongoDB", "id", paste.ID, "size", paste.Size)
	return nil
}

// Get retrieves a paste from MongoDB
func (m *MongoDBStorage) Get(id string) (*Paste, error) {
	if m.client == nil {
		return nil, fmt.Errorf("MongoDB client not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	coll := m.client.Database(m.database).Collection(m.collection)

	var doc bson.M
	err := coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Paste not found
		}
		m.logger.Error("Failed to get paste", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get paste: %w", err)
	}

	// Convert from MongoDB document to Paste
	paste := &Paste{
		ID:          getStringFromDoc(doc, "_id"),
		Content:     getBytesFromDoc(doc, "content"),
		ContentType: getStringFromDoc(doc, "content_type"),
		Filename:    getStringFromDoc(doc, "filename"),
		Language:    getStringFromDoc(doc, "language"),
		Title:       getStringFromDoc(doc, "title"),
		CreatedAt:   getTimeValueFromDoc(doc, "created_at"),
		ExpiresAt:   getTimeFromDoc(doc, "expires_at"),
		ClientIP:    getStringFromDoc(doc, "client_ip"),
		Size:        getInt64FromDoc(doc, "size"),
		Metadata:    getMetadataFromDoc(doc, "metadata"),
	}

	return paste, nil
}

// Helper functions to safely extract values from BSON documents
func getStringFromDoc(doc bson.M, key string) string {
	if val, ok := doc[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getBytesFromDoc(doc bson.M, key string) []byte {
	if val, ok := doc[key]; ok {
		// Since we're storing content as string, convert back to bytes
		if str, ok := val.(string); ok {
			return []byte(str)
		}
		// Also handle if it was stored as bytes
		if bytes, ok := val.([]byte); ok {
			return bytes
		}
		// Handle primitive.Binary type (from BSON)
		if binary, ok := val.(primitive.Binary); ok {
			return binary.Data
		}
	}
	return nil
}

func getTimeFromDoc(doc bson.M, key string) *time.Time {
	if val, ok := doc[key]; ok {
		if t, ok := val.(time.Time); ok {
			return &t
		}
	}
	return nil
}

func getTimeValueFromDoc(doc bson.M, key string) time.Time {
	if val, ok := doc[key]; ok {
		if t, ok := val.(time.Time); ok {
			return t
		}
	}
	return time.Time{}
}

func getInt64FromDoc(doc bson.M, key string) int64 {
	if val, ok := doc[key]; ok {
		switch v := val.(type) {
		case int64:
			return v
		case int32:
			return int64(v)
		case int:
			return int64(v)
		}
	}
	return 0
}

func getMetadataFromDoc(doc bson.M, key string) map[string]string {
	if val, ok := doc[key]; ok {
		if metadata, ok := val.(bson.M); ok {
			result := make(map[string]string)
			for k, v := range metadata {
				if str, ok := v.(string); ok {
					result[k] = str
				}
			}
			return result
		}
	}
	return nil
}

// Exists checks if a paste exists in MongoDB
func (m *MongoDBStorage) Exists(id string) bool {
	if m.client == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	coll := m.client.Database(m.database).Collection(m.collection)

	count, err := coll.CountDocuments(ctx, bson.M{"_id": id})
	if err != nil {
		m.logger.Error("Failed to check paste existence", "id", id, "error", err)
		return false
	}

	return count > 0
}

// Delete removes a paste by ID
func (m *MongoDBStorage) Delete(id string) error {
	if m.client == nil {
		return fmt.Errorf("MongoDB client not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	coll := m.client.Database(m.database).Collection(m.collection)

	result, err := coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		m.logger.Error("Failed to delete paste", "id", id, "error", err)
		return fmt.Errorf("failed to delete paste: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("paste not found: %s", id)
	}

	m.logger.Debug("Paste deleted successfully", "id", id)
	return nil
}

// List returns a list of paste IDs
func (m *MongoDBStorage) List(limit int) ([]string, error) {
	if m.client == nil {
		return nil, fmt.Errorf("MongoDB client not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := m.client.Database(m.database).Collection(m.collection)

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetProjection(bson.M{"_id": 1})

	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := coll.Find(ctx, bson.M{}, opts)
	if err != nil {
		m.logger.Error("Failed to list pastes", "error", err)
		return nil, fmt.Errorf("failed to list pastes: %w", err)
	}
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			m.logger.Warn("Failed to close cursor in List", "error", err)
		}
	}()

	var ids []string
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			m.logger.Warn("Failed to decode paste document", "error", err)
			continue
		}
		if id, ok := doc["_id"].(string); ok {
			ids = append(ids, id)
		}
	}

	if err := cursor.Err(); err != nil {
		m.logger.Error("Cursor error while listing pastes", "error", err)
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return ids, nil
}

// Stats returns storage statistics
func (m *MongoDBStorage) Stats() (*Stats, error) {
	if m.client == nil {
		return &Stats{
			TotalPastes:   0,
			TotalSize:     0,
			ExpiredPastes: 0,
		}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := m.client.Database(m.database).Collection(m.collection)

	// Get total count
	totalCount, err := coll.CountDocuments(ctx, bson.M{})
	if err != nil {
		m.logger.Error("Failed to get total count", "error", err)
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	// Get total size
	pipeline := mongo.Pipeline{
		{primitive.E{Key: "$group", Value: bson.D{
			primitive.E{Key: "_id", Value: nil},
			primitive.E{Key: "totalSize", Value: bson.D{primitive.E{Key: "$sum", Value: "$size"}}},
		}}},
	}

	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		m.logger.Error("Failed to aggregate size", "error", err)
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			m.logger.Warn("Failed to close cursor in Stats", "error", err)
		}
	}()

	var totalSize int64
	if cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err == nil {
			if size, ok := result["totalSize"]; ok {
				if sizeInt, ok := size.(int64); ok {
					totalSize = sizeInt
				}
			}
		}
	}

	// Count expired pastes (optional - TTL should handle this automatically)
	expiredCount, _ := coll.CountDocuments(ctx, bson.M{
		"expires_at": bson.M{"$lt": time.Now()},
	})

	return &Stats{
		TotalPastes:   totalCount,
		TotalSize:     totalSize,
		ExpiredPastes: expiredCount,
	}, nil
}

// Cleanup manually removes expired pastes
func (m *MongoDBStorage) Cleanup() error {
	if m.client == nil {
		return nil
	}

	// MongoDB TTL index handles automatic cleanup
	m.logger.Info("MongoDB TTL handles automatic cleanup")

	// Optionally, we can manually remove already expired documents
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	coll := m.client.Database(m.database).Collection(m.collection)

	result, err := coll.DeleteMany(ctx, bson.M{
		"expires_at": bson.M{"$lt": time.Now()},
	})
	if err != nil {
		m.logger.Error("Failed to cleanup expired pastes", "error", err)
		return fmt.Errorf("failed to cleanup: %w", err)
	}

	if result.DeletedCount > 0 {
		m.logger.Info("Manually cleaned up expired pastes", "count", result.DeletedCount)
	}

	return nil
}

// Close closes the MongoDB connection
func (m *MongoDBStorage) Close() error {
	if m.client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := m.client.Disconnect(ctx)
	if err != nil {
		m.logger.Error("Failed to close MongoDB connection", "error", err)
		return fmt.Errorf("failed to close connection: %w", err)
	}

	m.logger.Info("MongoDB connection closed")
	return nil
}

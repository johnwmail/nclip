package storage

import (
	"context"
	"time"

	"github.com/johnwmail/nclip/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoStore implements PasteStore using MongoDB
type MongoStore struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
}

// NewMongoStore creates a new MongoDB storage backend
func NewMongoStore(url, dbName string) (*MongoStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(url))
	if err != nil {
		return nil, err
	}

	// Test the connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	database := client.Database(dbName)
	collection := database.Collection("pastes")

	store := &MongoStore{
		client:     client,
		database:   database,
		collection: collection,
	}

	// Create TTL index for auto-expiration
	if err := store.createIndexes(); err != nil {
		return nil, err
	}

	return store, nil
}

// createIndexes creates necessary indexes for the collection
func (m *MongoStore) createIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// TTL index on expires_at field
	ttlIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	}

	// Index on created_at for queries
	createdAtIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "created_at", Value: -1}},
	}

	_, err := m.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		ttlIndex,
		createdAtIndex,
	})

	return err
}

// Store saves a paste to MongoDB
func (m *MongoStore) Store(paste *models.Paste) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := m.collection.InsertOne(ctx, paste)
	return err
}

// Get retrieves a paste by its ID
func (m *MongoStore) Get(id string) (*models.Paste, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var paste models.Paste
	err := m.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&paste)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Not found
		}
		return nil, err
	}

	// Check if expired
	if paste.IsExpired() {
		// Delete expired paste
		m.Delete(id)
		return nil, nil
	}

	return &paste, nil
}

// Delete removes a paste from MongoDB
func (m *MongoStore) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := m.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// IncrementReadCount increments the read count for a paste
func (m *MongoStore) IncrementReadCount(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := m.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$inc": bson.M{"read_count": 1}},
	)
	return err
}

// Close closes the MongoDB connection
func (m *MongoStore) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return m.client.Disconnect(ctx)
}

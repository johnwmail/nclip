package storage

import (
	"fmt"
	"log/slog"

	"github.com/johnwmail/nclip/internal/config"
)

// NewStorage creates a storage backend based on the configuration
func NewStorage(cfg *config.Config, logger *slog.Logger) (Storage, error) {
	switch cfg.StorageType {
	case "filesystem":
		return NewFilesystemStorage(cfg.OutputDir)

	case "mongodb":
		// MongoDB storage - production ready
		logger.Info("Using MongoDB storage",
			"uri", cfg.MongoDBURI,
			"database", cfg.MongoDBDatabase,
			"collection", cfg.MongoDBCollection)

		storage := NewMongoDBStorage(cfg.MongoDBURI, cfg.MongoDBDatabase, cfg.MongoDBCollection, logger)
		return storage, nil

	case "dynamodb":
		// DynamoDB storage - template implementation
		logger.Warn("DynamoDB storage is not fully implemented yet",
			"table", cfg.DynamoDBTable)

		storage, err := NewDynamoDBStorage(cfg.DynamoDBTable, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create DynamoDB storage: %w", err)
		}
		return storage, nil

	default:
		return nil, fmt.Errorf("unsupported storage type: %s (supported: filesystem, mongodb, dynamodb)", cfg.StorageType)
	}
}

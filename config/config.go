package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the nclip service
type Config struct {
	Port          int           `json:"port"`
	URL           string        `json:"url"`
	SlugLength    int           `json:"slug_length"`
	BufferSize    int64         `json:"buffer_size"`
	DefaultTTL    time.Duration `json:"default_ttl"`
	EnableMetrics bool          `json:"enable_metrics"`
	EnableWebUI   bool          `json:"enable_webui"`
	StorageType   string        `json:"storage_type"`
	MongoURL      string        `json:"mongo_url"`
	DynamoTable   string        `json:"dynamo_table"`
	DynamoRegion  string        `json:"dynamo_region"`
}

// LoadConfig loads configuration from environment variables and CLI flags
func LoadConfig() *Config {
	config := &Config{
		Port:          8080,
		URL:           "",
		SlugLength:    5,
		BufferSize:    1048576, // 1MB
		DefaultTTL:    24 * time.Hour,
		EnableMetrics: true,
		EnableWebUI:   true,
		StorageType:   "mongodb",
		MongoURL:      "mongodb://localhost:27017",
		DynamoTable:   "nclip-pastes",
		DynamoRegion:  "us-east-1",
	}

	// Parse CLI flags
	flag.IntVar(&config.Port, "port", config.Port, "Port to listen on")
	flag.StringVar(&config.URL, "url", config.URL, "Base URL for paste links")
	flag.IntVar(&config.SlugLength, "slug-length", config.SlugLength, "Length of generated slugs")
	flag.Int64Var(&config.BufferSize, "buffer-size", config.BufferSize, "Maximum upload size in bytes")
	flag.DurationVar(&config.DefaultTTL, "ttl", config.DefaultTTL, "Default paste expiration time")
	flag.BoolVar(&config.EnableMetrics, "enable-metrics", config.EnableMetrics, "Enable Prometheus metrics")
	flag.BoolVar(&config.EnableWebUI, "enable-webui", config.EnableWebUI, "Enable web UI")
	flag.StringVar(&config.StorageType, "storage-type", config.StorageType, "Storage backend (mongodb or dynamodb)")
	flag.StringVar(&config.MongoURL, "mongo-url", config.MongoURL, "MongoDB connection URL")
	flag.StringVar(&config.DynamoTable, "dynamo-table", config.DynamoTable, "DynamoDB table name")
	flag.StringVar(&config.DynamoRegion, "dynamo-region", config.DynamoRegion, "DynamoDB region")
	flag.Parse()

	// Override with environment variables if present
	if val := os.Getenv("NCLIP_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.Port = port
		}
	}
	if val := os.Getenv("NCLIP_URL"); val != "" {
		config.URL = val
	}
	if val := os.Getenv("NCLIP_SLUG_LENGTH"); val != "" {
		if length, err := strconv.Atoi(val); err == nil {
			config.SlugLength = length
		}
	}
	if val := os.Getenv("NCLIP_BUFFER_SIZE"); val != "" {
		if size, err := strconv.ParseInt(val, 10, 64); err == nil {
			config.BufferSize = size
		}
	}
	if val := os.Getenv("NCLIP_TTL"); val != "" {
		if ttl, err := time.ParseDuration(val); err == nil {
			config.DefaultTTL = ttl
		}
	}
	if val := os.Getenv("NCLIP_ENABLE_METRICS"); val != "" {
		config.EnableMetrics = val == "true"
	}
	if val := os.Getenv("NCLIP_ENABLE_WEBUI"); val != "" {
		config.EnableWebUI = val == "true"
	}
	if val := os.Getenv("NCLIP_STORAGE_TYPE"); val != "" {
		config.StorageType = val
	}
	if val := os.Getenv("NCLIP_MONGO_URL"); val != "" {
		config.MongoURL = val
	}
	if val := os.Getenv("NCLIP_DYNAMO_TABLE"); val != "" {
		config.DynamoTable = val
	}
	if val := os.Getenv("NCLIP_DYNAMO_REGION"); val != "" {
		config.DynamoRegion = val
	}

	return config
}

package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the nclip service
type Config struct {
	Port        int           `json:"port"`
	URL         string        `json:"url"`
	SlugLength  int           `json:"slug_length"`
	BufferSize  int64         `json:"buffer_size"`
	DefaultTTL  time.Duration `json:"default_ttl"`
	HTTPSOnly   bool          `json:"https_only"`
	MongoURL    string        `json:"mongo_url"`
	DynamoTable string        `json:"dynamo_table"`
	Version     string        `json:"version"`
	BuildTime   string        `json:"build_time"`
	CommitHash  string        `json:"commit_hash"`
}

// LoadConfig loads configuration from environment variables and CLI flags
func LoadConfig() *Config {
	config := &Config{
		Port:        8080,
		URL:         "",
		SlugLength:  5,
		BufferSize:  1048576, // 1MB
		DefaultTTL:  24 * time.Hour,
		HTTPSOnly:   false,
		MongoURL:    "mongodb://localhost:27017",
		DynamoTable: "nclip-pastes",
	}

	// Parse CLI flags
	flag.IntVar(&config.Port, "port", config.Port, "Port to listen on")
	flag.StringVar(&config.URL, "url", config.URL, "Base URL for paste links")
	flag.IntVar(&config.SlugLength, "slug-length", config.SlugLength, "Length of generated slugs")
	flag.Int64Var(&config.BufferSize, "buffer-size", config.BufferSize, "Maximum upload size in bytes")
	flag.DurationVar(&config.DefaultTTL, "ttl", config.DefaultTTL, "Default paste expiration time")
	flag.BoolVar(&config.HTTPSOnly, "https-only", config.HTTPSOnly, "Force HTTPS URLs when base URL is not set")
	flag.StringVar(&config.MongoURL, "mongo-url", config.MongoURL, "MongoDB connection URL")
	flag.StringVar(&config.DynamoTable, "dynamo-table", config.DynamoTable, "DynamoDB table name")
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
	if val := os.Getenv("NCLIP_HTTPS_ONLY"); val != "" {
		config.HTTPSOnly = val == "true"
	}
	if val := os.Getenv("NCLIP_MONGO_URL"); val != "" {
		config.MongoURL = val
	}
	if val := os.Getenv("NCLIP_DYNAMO_TABLE"); val != "" {
		config.DynamoTable = val
	}

	return config
}

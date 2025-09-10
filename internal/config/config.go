package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration options for the nclip server
type Config struct {
	// Server configuration
	Domain   string
	TCPPort  int
	HTTPPort int

	// Storage configuration
	StorageType string // "filesystem", "mongodb", "dynamodb"
	OutputDir   string
	SlugLength  int
	BufferSize  int64

	// Database configuration
	MongoDBURI        string
	MongoDBDatabase   string
	MongoDBCollection string
	DynamoDBTable     string

	// Paste configuration
	ExpireDays int
	RateLimit  string

	// Operational configuration
	LogLevel string
	UserName string
	LogFile  string

	// Feature flags
	EnableWebUI   bool
	EnableMetrics bool
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Domain:            "localhost",
		TCPPort:           9999,
		HTTPPort:          8080,
		StorageType:       "filesystem",
		OutputDir:         "./pastes",
		SlugLength:        8,
		BufferSize:        1024 * 1024, // 1MB
		MongoDBURI:        "mongodb://localhost:27017",
		MongoDBDatabase:   "nclip",
		MongoDBCollection: "pastes",
		DynamoDBTable:     "nclip-pastes",
		ExpireDays:        1, // 1 day default for serverless
		RateLimit:         "10/min",
		LogLevel:          "info",
		EnableWebUI:       true,
		EnableMetrics:     true,
	}
}

// LoadFromFlags parses command-line flags and environment variables
func LoadFromFlags() (*Config, error) {
	cfg := DefaultConfig()

	// Define command-line flags
	flag.StringVar(&cfg.Domain, "domain", getEnvString("NCLIP_DOMAIN", cfg.Domain), "Domain name for generated URLs")
	flag.IntVar(&cfg.TCPPort, "tcp-port", getEnvInt("NCLIP_TCP_PORT", cfg.TCPPort), "TCP port for netcat connections")
	flag.IntVar(&cfg.HTTPPort, "http-port", getEnvInt("NCLIP_HTTP_PORT", cfg.HTTPPort), "HTTP port for web interface")

	// Storage configuration
	flag.StringVar(&cfg.StorageType, "storage-type", getEnvString("NCLIP_STORAGE_TYPE", cfg.StorageType), "Storage backend: filesystem, mongodb, dynamodb")
	flag.StringVar(&cfg.OutputDir, "output-dir", getEnvString("NCLIP_OUTPUT_DIR", cfg.OutputDir), "Directory to store paste files (filesystem only)")
	flag.IntVar(&cfg.SlugLength, "slug-length", getEnvInt("NCLIP_SLUG_LENGTH", cfg.SlugLength), "Length of generated slug IDs")

	// Database configuration
	flag.StringVar(&cfg.MongoDBURI, "mongodb-uri", getEnvString("NCLIP_MONGODB_URI", cfg.MongoDBURI), "MongoDB connection URI")
	flag.StringVar(&cfg.MongoDBDatabase, "mongodb-database", getEnvString("NCLIP_MONGODB_DATABASE", cfg.MongoDBDatabase), "MongoDB database name")
	flag.StringVar(&cfg.MongoDBCollection, "mongodb-collection", getEnvString("NCLIP_MONGODB_COLLECTION", cfg.MongoDBCollection), "MongoDB collection name")
	flag.StringVar(&cfg.DynamoDBTable, "dynamodb-table", getEnvString("NCLIP_DYNAMODB_TABLE", cfg.DynamoDBTable), "DynamoDB table name")

	var bufferSizeMB int
	flag.IntVar(&bufferSizeMB, "buffer-size-mb", int(cfg.BufferSize/(1024*1024)), "Maximum paste size in MB")

	flag.IntVar(&cfg.ExpireDays, "expire-days", getEnvInt("NCLIP_EXPIRE_DAYS", cfg.ExpireDays), "Auto-delete pastes after N days")
	flag.StringVar(&cfg.RateLimit, "rate-limit", getEnvString("NCLIP_RATE_LIMIT", cfg.RateLimit), "Rate limit per IP (e.g., 10/min)")

	flag.StringVar(&cfg.LogLevel, "log-level", getEnvString("NCLIP_LOG_LEVEL", cfg.LogLevel), "Log level (debug, info, warn, error)")
	flag.StringVar(&cfg.UserName, "user", getEnvString("NCLIP_USER", cfg.UserName), "User to run as (requires root)")
	flag.StringVar(&cfg.LogFile, "log-file", getEnvString("NCLIP_LOG_FILE", cfg.LogFile), "Path to log file")

	flag.BoolVar(&cfg.EnableWebUI, "enable-webui", getEnvBool("NCLIP_ENABLE_WEBUI", cfg.EnableWebUI), "Enable web UI")
	flag.BoolVar(&cfg.EnableMetrics, "enable-metrics", getEnvBool("NCLIP_ENABLE_METRICS", cfg.EnableMetrics), "Enable metrics endpoint")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "nclip - Modern netcat-to-clipboard service\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  All flags can be set via environment variables with NCLIP_ prefix\n")
		fmt.Fprintf(os.Stderr, "  Example: NCLIP_DOMAIN=example.com\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  # Start with default settings\n")
		fmt.Fprintf(os.Stderr, "  %s\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Custom domain and ports\n")
		fmt.Fprintf(os.Stderr, "  %s -domain paste.example.com -tcp-port 9999 -http-port 8080\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # With custom storage and directory\n")
		fmt.Fprintf(os.Stderr, "  %s -domain paste.example.com -storage-type mongodb -mongodb-uri mongodb://localhost:27017\n\n", os.Args[0])
	}

	flag.Parse()

	// Convert buffer size from MB to bytes
	if bufferSizeMB > 0 {
		cfg.BufferSize = int64(bufferSizeMB) * 1024 * 1024
	}

	return cfg, cfg.Validate()
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	if c.TCPPort < 1 || c.TCPPort > 65535 {
		return fmt.Errorf("invalid TCP port: %d", c.TCPPort)
	}

	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", c.HTTPPort)
	}

	if c.TCPPort == c.HTTPPort {
		return fmt.Errorf("TCP and HTTP ports cannot be the same: %d", c.TCPPort)
	}

	if c.SlugLength < 1 || c.SlugLength > 32 {
		return fmt.Errorf("slug length must be between 1 and 32: %d", c.SlugLength)
	}

	if c.BufferSize < 1024 || c.BufferSize > 100*1024*1024 {
		return fmt.Errorf("buffer size must be between 1KB and 100MB: %d", c.BufferSize)
	}

	if c.ExpireDays < 0 {
		return fmt.Errorf("expire days cannot be negative: %d", c.ExpireDays)
	}

	// Validate storage type
	validStorageTypes := []string{"filesystem", "mongodb", "dynamodb"}
	validType := false
	for _, st := range validStorageTypes {
		if c.StorageType == st {
			validType = true
			break
		}
	}
	if !validType {
		return fmt.Errorf("invalid storage type: %s (valid: filesystem, mongodb, dynamodb)", c.StorageType)
	}

	return nil
}

// GetBaseURL returns the base URL for paste links
// Note: HTTPS/TLS should be handled by reverse proxy (nginx, HAProxy, etc.)
func (c *Config) GetBaseURL() string {
	// Always use http - reverse proxy handles HTTPS termination
	scheme := "http"

	// Don't include port in URL if using standard HTTP port
	if c.HTTPPort == 80 {
		return fmt.Sprintf("%s://%s", scheme, c.Domain)
	}

	return fmt.Sprintf("%s://%s:%d", scheme, c.Domain, c.HTTPPort)
}

// GetExpiration returns the expiration duration
func (c *Config) GetExpiration() time.Duration {
	if c.ExpireDays == 0 {
		return 0 // No expiration
	}
	return time.Duration(c.ExpireDays) * 24 * time.Hour
}

// Helper functions for environment variable parsing
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

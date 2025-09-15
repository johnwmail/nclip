package config

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration options for the nclip server
type Config struct {
	// Server configuration
	BaseURL  string
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
		// Server defaults
		BaseURL:           "http://localhost:8080/",
		TCPPort:           8099,
		HTTPPort:          8080,
		StorageType:       "filesystem",
		OutputDir:         "./pastes",
		SlugLength:        5,
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
	flag.StringVar(&cfg.BaseURL, "url", getEnvString("NCLIP_URL", cfg.BaseURL), "Base URL template for generated paste URLs (e.g., https://paste.example.com/clips/)")
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
		fmt.Fprintf(os.Stderr, "  Example: NCLIP_URL=https://paste.example.com/clips/\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  # Start with default settings\n")
		fmt.Fprintf(os.Stderr, "  %s\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Custom URL and ports\n")
		fmt.Fprintf(os.Stderr, "  %s -url https://paste.example.com/clips/ -tcp-port 8099 -http-port 8080\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # With custom storage and directory\n")
		fmt.Fprintf(os.Stderr, "  %s -url https://paste.example.com/clips/ -storage-type mongodb -mongodb-uri mongodb://localhost:27017\n\n", os.Args[0])
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
	if c.BaseURL == "" {
		return fmt.Errorf("base URL cannot be empty")
	}

	// Validate URL format
	if !isValidURL(c.BaseURL) {
		return fmt.Errorf("invalid base URL format: %s", c.BaseURL)
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
// The BaseURL is now a complete URL template that includes protocol, domain, port, and path
func (c *Config) GetBaseURL() string {
	return c.BaseURL
}

// JoinBaseURLAndSlug joins the base URL and slug, ensuring exactly one slash between them
func JoinBaseURLAndSlug(baseURL, slug string) string {
	if strings.HasSuffix(baseURL, "/") {
		return baseURL + slug
	}
	return baseURL + "/" + slug
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

// isValidURL validates if the provided string is a valid HTTP/HTTPS URL
func isValidURL(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	// Must have a scheme (protocol)
	if parsedURL.Scheme == "" {
		return false
	}

	// Must be http or https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false
	}

	// Must have a host
	if parsedURL.Host == "" {
		return false
	}

	return true
}

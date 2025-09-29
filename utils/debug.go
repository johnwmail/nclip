package utils

import "os"

// IsDebugEnabled returns true unless GIN_MODE=release
func IsDebugEnabled() bool {
	return os.Getenv("GIN_MODE") != "release"
}

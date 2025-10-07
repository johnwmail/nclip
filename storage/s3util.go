package storage

import "strings"

func applyS3Prefix(prefix, name string) string {
	if prefix == "" {
		return name
	}
	// Ensure there is exactly one slash between prefix and name
	if strings.HasSuffix(prefix, "/") {
		return prefix + name
	}
	return prefix + "/" + name
}

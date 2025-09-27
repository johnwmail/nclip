package storage

import "strings"

func normalizeS3Prefix(prefix string) string {
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return ""
	}
	return prefix + "/"
}

func applyS3Prefix(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + name
}

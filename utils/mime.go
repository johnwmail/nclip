package utils

import (
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

// DetectContentType attempts to detect the MIME type of content
// It first tries to detect from the filename, then from the content itself
func DetectContentType(filename string, content []byte) string {
	// Try to detect from filename extension first
	if filename != "" {
		ext := strings.ToLower(filepath.Ext(filename))
		if mimeType := mime.TypeByExtension(ext); mimeType != "" {
			return mimeType
		}
	}

	// Fallback to content-based detection
	if len(content) > 0 {
		return http.DetectContentType(content)
	}

	// Default fallback
	return "application/octet-stream"
}

// IsTextContent returns true if the content type is text-based
func IsTextContent(contentType string) bool {
	textTypes := []string{
		"text/",
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-sh",
		"application/x-yaml",
	}

	contentType = strings.ToLower(contentType)
	for _, textType := range textTypes {
		if strings.HasPrefix(contentType, textType) {
			return true
		}
	}

	return false
}

// ExtensionByMime returns the most common file extension for a given MIME type
func ExtensionByMime(mimeType string) string {
	if mimeType == "" {
		return ""
	}
	exts, _ := mime.ExtensionsByType(mimeType)
	if len(exts) > 0 {
		return exts[0]
	}
	return ""
}

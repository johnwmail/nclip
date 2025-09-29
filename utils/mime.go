package utils

import (
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

// extensionMap holds common mime type to extension mappings
var extensionMap = map[string]string{
	"application/zip":              ".zip",
	"application/x-zip-compressed": ".zip",
	"application/x-zip":            ".zip",
	"application/x-tar":            ".tar",
	"application/tar":              ".tar",
	"application/x-gzip":           ".gz",
	"application/gzip":             ".gz",
	"application/x-7z-compressed":  ".7z",
	"application/7z":               ".7z",
	"application/x-bzip2":          ".bz2",
	"application/bzip2":            ".bz2",
	"application/x-xz":             ".xz",
	"application/xz":               ".xz",
	"application/x-rar-compressed": ".rar",
	"application/vnd.rar":          ".rar",
	"application/rar":              ".rar",
	"application/pdf":              ".pdf",
	"image/jpeg":                   ".jpg",
	"image/png":                    ".png",
	"image/gif":                    ".gif",
	"image/webp":                   ".webp",
	"image/svg+xml":                ".svg",
	"application/octet-stream":     ".bin",
	"application/x-binary":         ".bin",
	"application/bin":              ".bin",
}

func extensionByMimeMap(mimeType string) string {
	ext, ok := extensionMap[strings.ToLower(mimeType)]
	if ok {
		return ext
	}
	return ""
}

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
	if base, _, err := mime.ParseMediaType(mimeType); err == nil {
		mimeType = base
	}
	if exts, _ := mime.ExtensionsByType(mimeType); len(exts) > 0 && exts[0] != "" {
		return exts[0]
	}
	return extensionByMimeMap(mimeType)
}

package utils

import (
	"bytes"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

// extensionMap holds common mime type to extension mappings
var extensionMap = map[string]string{
	// Archives and compressed files
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

	// Documents and media
	"application/pdf": ".pdf",
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/gif":       ".gif",
	"image/webp":      ".webp",
	"image/svg+xml":   ".svg",

	// Text files - user-friendly extensions
	"text/plain":      ".txt",
	"text/html":       ".html",
	"text/css":        ".css",
	"text/javascript": ".js",
	"text/xml":        ".xml",
	"text/markdown":   ".md",
	"text/x-python":   ".py",
	"text/x-go":       ".go",
	"text/x-sh":       ".sh",
	"text/x-yaml":     ".yaml",
	"text/x-toml":     ".toml",

	// Application types
	"application/json":          ".json",
	"application/xml":           ".xml",
	"application/yaml":          ".yaml",
	"application/x-yaml":        ".yaml",
	"application/toml":          ".toml",
	"application/x-toml":        ".toml",
	"application/javascript":    ".js",
	"application/x-sh":          ".sh",
	"application/x-python-code": ".py",

	// Binary and generic
	"application/octet-stream": ".bin",
	"application/x-binary":     ".bin",
	"application/bin":          ".bin",
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

	// Fallback to content-based detection using helper
	if len(content) > 0 {
		if mt := detectByMagic(content); mt != "" {
			return mt
		}
		return http.DetectContentType(content)
	}

	// Default fallback
	return "application/octet-stream"
}

// detectByMagic checks common file signatures (magic numbers) and
// returns a mime type string if recognized, or empty string otherwise.
func detectByMagic(content []byte) string {
	// Table-driven signature checks reduce branching and cyclomatic complexity.
	var signatures = []struct {
		sig  []byte
		mime string
	}{
		{[]byte{'P', 'K', 0x03, 0x04}, "application/zip"},
		{[]byte{0x89, 'P', 'N', 'G'}, "image/png"},
		{[]byte{0xFF, 0xD8, 0xFF}, "image/jpeg"},
		{[]byte("GIF87a"), "image/gif"},
		{[]byte("GIF89a"), "image/gif"},
	}

	for _, s := range signatures {
		if len(content) >= len(s.sig) && bytes.Equal(content[:len(s.sig)], s.sig) {
			return s.mime
		}
	}

	return ""
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

	// Check our custom extension map first for user-friendly extensions
	if ext := extensionByMimeMap(mimeType); ext != "" {
		return ext
	}

	// Fall back to Go's built-in mime package
	if exts, _ := mime.ExtensionsByType(mimeType); len(exts) > 0 && exts[0] != "" {
		return exts[0]
	}

	return ""
}

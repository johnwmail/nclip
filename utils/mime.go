package utils

import (
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

	// Content-based detection
	if len(content) > 0 {
		// Check for ZIP file signature (PK\x03\x04)
		if len(content) >= 4 && content[0] == 'P' && content[1] == 'K' && content[2] == 0x03 && content[3] == 0x04 {
			return "application/zip"
		}

		// Check for other common file signatures
		// PNG: \x89PNG
		if len(content) >= 4 && content[0] == 0x89 && content[1] == 'P' && content[2] == 'N' && content[3] == 'G' {
			return "image/png"
		}
		// JPEG: \xFF\xD8\xFF
		if len(content) >= 3 && content[0] == 0xFF && content[1] == 0xD8 && content[2] == 0xFF {
			return "image/jpeg"
		}
		// GIF: GIF87a or GIF89a
		if len(content) >= 6 && content[0] == 'G' && content[1] == 'I' && content[2] == 'F' && content[3] == '8' && (content[4] == '7' || content[4] == '9') && content[5] == 'a' {
			return "image/gif"
		}

		// Fallback to http.DetectContentType
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

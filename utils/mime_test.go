package utils

import (
	"testing"
)

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  []byte
		want     string
	}{
		{
			name:     "detect from filename - text",
			filename: "test.txt",
			content:  []byte("hello world"),
			want:     "text/plain; charset=utf-8",
		},
		{
			name:     "detect from filename - json",
			filename: "config.json",
			content:  []byte(`{"key": "value"}`),
			want:     "application/json",
		},
		{
			name:     "detect from filename - zip",
			filename: "archive.zip",
			content:  []byte("PK\x03\x04"), // ZIP header
			want:     "application/zip",
		},
		{
			name:     "detect from content - html",
			filename: "",
			content:  []byte("<html><body>test</body></html>"),
			want:     "text/html; charset=utf-8",
		},
		{
			name:     "detect from content - binary",
			filename: "",
			content:  []byte{0x00, 0x01, 0x02, 0x03},
			want:     "application/octet-stream",
		},
		{
			name:     "empty content and filename",
			filename: "",
			content:  []byte{},
			want:     "application/octet-stream",
		},
		{
			name:     "filename without extension",
			filename: "README",
			content:  []byte("This is a readme file"),
			want:     "text/plain; charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectContentType(tt.filename, tt.content)
			if got != tt.want {
				t.Errorf("DetectContentType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsTextContent(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        bool
	}{
		{
			name:        "plain text",
			contentType: "text/plain",
			want:        true,
		},
		{
			name:        "html text",
			contentType: "text/html",
			want:        true,
		},
		{
			name:        "json",
			contentType: "application/json",
			want:        true,
		},
		{
			name:        "xml",
			contentType: "application/xml",
			want:        true,
		},
		{
			name:        "javascript",
			contentType: "application/javascript",
			want:        true,
		},
		{
			name:        "shell script",
			contentType: "application/x-sh",
			want:        true,
		},
		{
			name:        "yaml",
			contentType: "application/x-yaml",
			want:        true,
		},
		{
			name:        "binary - image",
			contentType: "image/png",
			want:        false,
		},
		{
			name:        "binary - zip",
			contentType: "application/zip",
			want:        false,
		},
		{
			name:        "binary - pdf",
			contentType: "application/pdf",
			want:        false,
		},
		{
			name:        "case insensitive",
			contentType: "TEXT/PLAIN",
			want:        true,
		},
		{
			name:        "with charset",
			contentType: "text/plain; charset=utf-8",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTextContent(tt.contentType)
			if got != tt.want {
				t.Errorf("IsTextContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtensionByMime(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		want     string
	}{
		// Archives and compressed files
		{
			name:     "zip",
			mimeType: "application/zip",
			want:     ".zip",
		},
		{
			name:     "zip compressed",
			mimeType: "application/x-zip-compressed",
			want:     ".zip",
		},
		{
			name:     "zip x-zip",
			mimeType: "application/x-zip",
			want:     ".zip",
		},
		{
			name:     "tar",
			mimeType: "application/x-tar",
			want:     ".tar",
		},
		{
			name:     "tar application/tar",
			mimeType: "application/tar",
			want:     ".tar",
		},
		{
			name:     "gzip",
			mimeType: "application/x-gzip",
			want:     ".gz",
		},
		{
			name:     "gzip application/gzip",
			mimeType: "application/gzip",
			want:     ".gz",
		},
		{
			name:     "7z",
			mimeType: "application/x-7z-compressed",
			want:     ".7z",
		},
		{
			name:     "7z application/7z",
			mimeType: "application/7z",
			want:     ".7z",
		},
		{
			name:     "bzip2",
			mimeType: "application/x-bzip2",
			want:     ".bz2",
		},
		{
			name:     "bzip2 application/bzip2",
			mimeType: "application/bzip2",
			want:     ".bz2",
		},
		{
			name:     "xz",
			mimeType: "application/x-xz",
			want:     ".xz",
		},
		{
			name:     "xz application/xz",
			mimeType: "application/xz",
			want:     ".xz",
		},
		{
			name:     "rar",
			mimeType: "application/x-rar-compressed",
			want:     ".rar",
		},
		{
			name:     "rar vnd.rar",
			mimeType: "application/vnd.rar",
			want:     ".rar",
		},
		{
			name:     "rar application/rar",
			mimeType: "application/rar",
			want:     ".rar",
		},

		// Documents and media
		{
			name:     "pdf",
			mimeType: "application/pdf",
			want:     ".pdf",
		},
		{
			name:     "jpeg image",
			mimeType: "image/jpeg",
			want:     ".jpg",
		},
		{
			name:     "png image",
			mimeType: "image/png",
			want:     ".png",
		},
		{
			name:     "gif image",
			mimeType: "image/gif",
			want:     ".gif",
		},
		{
			name:     "webp image",
			mimeType: "image/webp",
			want:     ".webp",
		},
		{
			name:     "svg image",
			mimeType: "image/svg+xml",
			want:     ".svg",
		},

		// Text files - user-friendly extensions
		{
			name:     "text plain",
			mimeType: "text/plain",
			want:     ".txt",
		},
		{
			name:     "html",
			mimeType: "text/html",
			want:     ".html",
		},
		{
			name:     "css",
			mimeType: "text/css",
			want:     ".css",
		},
		{
			name:     "javascript text",
			mimeType: "text/javascript",
			want:     ".js",
		},
		{
			name:     "xml text",
			mimeType: "text/xml",
			want:     ".xml",
		},
		{
			name:     "markdown",
			mimeType: "text/markdown",
			want:     ".md",
		},
		{
			name:     "python text",
			mimeType: "text/x-python",
			want:     ".py",
		},
		{
			name:     "go text",
			mimeType: "text/x-go",
			want:     ".go",
		},
		{
			name:     "shell script text",
			mimeType: "text/x-sh",
			want:     ".sh",
		},
		{
			name:     "yaml text",
			mimeType: "text/x-yaml",
			want:     ".yaml",
		},
		{
			name:     "toml text",
			mimeType: "text/x-toml",
			want:     ".toml",
		},

		// Application types
		{
			name:     "json",
			mimeType: "application/json",
			want:     ".json",
		},
		{
			name:     "xml application",
			mimeType: "application/xml",
			want:     ".xml",
		},
		{
			name:     "yaml application",
			mimeType: "application/yaml",
			want:     ".yaml",
		},
		{
			name:     "yaml x-yaml",
			mimeType: "application/x-yaml",
			want:     ".yaml",
		},
		{
			name:     "toml application",
			mimeType: "application/toml",
			want:     ".toml",
		},
		{
			name:     "toml x-toml",
			mimeType: "application/x-toml",
			want:     ".toml",
		},
		{
			name:     "javascript app",
			mimeType: "application/javascript",
			want:     ".js",
		},
		{
			name:     "shell script app",
			mimeType: "application/x-sh",
			want:     ".sh",
		},
		{
			name:     "python code app",
			mimeType: "application/x-python-code",
			want:     ".py",
		},

		// Binary and generic
		{
			name:     "octet stream",
			mimeType: "application/octet-stream",
			want:     ".bin",
		},
		{
			name:     "binary x-binary",
			mimeType: "application/x-binary",
			want:     ".bin",
		},
		{
			name:     "binary application/bin",
			mimeType: "application/bin",
			want:     ".bin",
		},

		// Edge cases
		{
			name:     "empty mime type",
			mimeType: "",
			want:     "",
		},
		{
			name:     "unknown mime type",
			mimeType: "application/x-unknown-type",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtensionByMime(tt.mimeType)
			if got != tt.want {
				t.Errorf("ExtensionByMime() = %v, want %v", got, tt.want)
			}
		})
	}
}

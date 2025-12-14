package server

import (
	"strings"
)

// staticFiles contains embedded static files for the dashboard
// This map is populated at build time or can be extended
var staticFiles = map[string]struct {
	Content     []byte
	ContentType string
}{}

// getStaticFile returns the content and content type for a static file
func getStaticFile(path string) (content []byte, contentType string, found bool) {
	// Normalize path
	path = strings.TrimPrefix(path, "/")

	file, ok := staticFiles[path]
	if !ok {
		return nil, "", false
	}

	return file.Content, file.ContentType, true
}

// RegisterStaticFile registers a static file to be served
func RegisterStaticFile(path, contentType string, content []byte) {
	path = strings.TrimPrefix(path, "/")
	staticFiles[path] = struct {
		Content     []byte
		ContentType string
	}{
		Content:     content,
		ContentType: contentType,
	}
}

// getContentType returns the content type for a file extension
func getContentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(path, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(path, ".js"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(path, ".json"):
		return "application/json; charset=utf-8"
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	case strings.HasSuffix(path, ".ico"):
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}

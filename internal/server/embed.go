package server

import (
	"embed"
	"io/fs"
	"path/filepath"
	"strings"
)

//go:embed all:public
var publicFS embed.FS

func init() {
	// Walk through all embedded files and register them
	fs.WalkDir(publicFS, "public", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Read file content
		content, err := publicFS.ReadFile(path)
		if err != nil {
			return err
		}

		// Convert path to URL path (remove "public" prefix)
		urlPath := strings.TrimPrefix(path, "public")
		if !strings.HasPrefix(urlPath, "/") {
			urlPath = "/" + urlPath
		}

		// Register the file
		contentType := getContentType(filepath.Base(path))
		RegisterStaticFile(urlPath, contentType, content)

		return nil
	})
}

// Package web provides the embedded web UI static files.
package web

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed dist/*
var distFS embed.FS

// FS returns the filesystem for serving the web UI.
// Returns an error if the dist directory doesn't exist (not built yet).
func FS() (fs.FS, error) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, fmt.Errorf("web UI not built: %w", err)
	}

	// Check if the directory has any content
	entries, err := fs.ReadDir(sub, ".")
	if err != nil || len(entries) == 0 {
		return nil, fmt.Errorf("web UI dist directory is empty; run 'make ui' first")
	}

	return sub, nil
}

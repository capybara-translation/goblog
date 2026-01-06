package http

import (
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/capybara-translation/goblog"
)

// HandleAdminSPA serves the React SPA for the admin interface
func HandleAdminSPA(w http.ResponseWriter, r *http.Request) {
	// Get the file system rooted at web-admin/dist
	spaFS, err := fs.Sub(goblog.AdminSPA, "web-admin/dist")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get the requested path (remove /admin prefix)
	path := strings.TrimPrefix(r.URL.Path, "/admin")
	if path == "" || path == "/" {
		path = "/index.html"
	}

	// Try to open the requested file
	file, err := spaFS.Open(strings.TrimPrefix(path, "/"))
	if err != nil {
		// File not found - serve index.html for client-side routing
		indexFile, err := spaFS.Open("index.html")
		if err != nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		defer indexFile.Close()

		// Get file info for size
		stat, err := indexFile.Stat()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Serve index.html
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, r, "index.html", stat.ModTime(), indexFile.(io.ReadSeeker))
		return
	}
	defer file.Close()

	// Get file info
	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// If it's a directory, serve index.html
	if stat.IsDir() {
		indexFile, err := spaFS.Open("index.html")
		if err != nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		defer indexFile.Close()

		indexStat, err := indexFile.Stat()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, r, "index.html", indexStat.ModTime(), indexFile.(io.ReadSeeker))
		return
	}

	// Set content type based on file extension
	ext := filepath.Ext(path)
	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	}

	// Serve the file
	http.ServeContent(w, r, stat.Name(), stat.ModTime(), file.(io.ReadSeeker))
}

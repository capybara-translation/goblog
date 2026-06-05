package http

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// DimensionsPrimer is the subset of *service.DiskDimensionsService that
// the upload handler depends on. Defining it locally lets us inject a
// fake in tests without importing the concrete service.
type DimensionsPrimer interface {
	Prime(url string, width, height int)
}

// ImageHandlers is a handler that processes image uploads
type ImageHandlers struct {
	uploadDir     string
	maxUploadSize int64
	primer        DimensionsPrimer // optional; nil disables cache priming
}

// NewImageHandlers creates a new ImageHandlers. primer may be nil; when
// non-nil, the handler reports width/height of each saved image so the
// public-page Markdown renderer can emit it without a first-render disk
// hit.
func NewImageHandlers(uploadDir string, maxUploadSize int64, primer DimensionsPrimer) *ImageHandlers {
	return &ImageHandlers{
		uploadDir:     uploadDir,
		maxUploadSize: maxUploadSize,
		primer:        primer,
	}
}

// ImageUploadResponse is the response when image upload succeeds
type ImageUploadResponse struct {
	URL      string `json:"url"`      // Image URL (e.g., /uploads/abc123.jpg)
	Filename string `json:"filename"` // Original filename
}

// Allowed MIME types and their extensions
var allowedMimeTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// Allowed file magic bytes (file signatures)
var allowedMagicBytes = map[string][]byte{
	"image/jpeg": {0xFF, 0xD8, 0xFF},
	"image/png":  {0x89, 0x50, 0x4E, 0x47},
	"image/gif":  {0x47, 0x49, 0x46, 0x38},
	"image/webp": {0x52, 0x49, 0x46, 0x46}, // RIFF header
}

// HandleUploadImage processes image uploads
// POST /api/v1/images
func (h *ImageHandlers) HandleUploadImage(w http.ResponseWriter, r *http.Request) {
	// Request body size limit
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadSize)

	// Parse multipart/form-data
	if err := r.ParseMultipartForm(h.maxUploadSize); err != nil {
		if err.Error() == "http: request body too large" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("File too large. Maximum size is %d bytes", h.maxUploadSize))
			return
		}
		writeError(w, http.StatusBadRequest, "Invalid form data")
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			r.MultipartForm.RemoveAll()
		}
	}()

	// Get the file
	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "No image file provided")
		return
	}
	defer file.Close()

	// Validate file size
	if header.Size > h.maxUploadSize {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("File too large. Maximum size is %d bytes", h.maxUploadSize))
		return
	}

	// Get and validate Content-Type
	contentType := header.Header.Get("Content-Type")
	ext, ok := allowedMimeTypes[contentType]
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid file type. Allowed types: JPEG, PNG, GIF, WebP")
		return
	}

	// Read file content (for magic byte validation)
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Printf("failed to read file: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to process image")
		return
	}

	// Validate magic bytes
	if !validateMagicBytes(fileBytes, contentType) {
		writeError(w, http.StatusBadRequest, "Invalid file content. File type does not match content")
		return
	}

	// Strip metadata (EXIF, etc.)
	strippedBytes, err := StripMetadata(fileBytes, contentType)
	if err != nil {
		log.Printf("failed to strip metadata from %s: %v", contentType, err)
		writeError(w, http.StatusBadRequest, "Failed to process image metadata")
		return
	}

	// Generate unique filename (UUID v4 + extension)
	newFilename := uuid.New().String() + ext

	// Build file path (prevent directory traversal)
	safePath := filepath.Join(h.uploadDir, filepath.Base(newFilename))

	// Save file (with metadata stripped)
	if err := os.WriteFile(safePath, strippedBytes, 0644); err != nil {
		log.Printf("failed to save file: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to save image")
		return
	}

	uploadURL := "/uploads/" + newFilename

	// Prime the dimensions cache so the first public-page render after an
	// upload doesn't trigger a disk hit. DecodeConfig reads only the
	// image header (microseconds-to-low-milliseconds), so this is cheap
	// relative to the rest of the upload pipeline. Failing here is a
	// silent regression of the CLS feature — the bytes passed
	// validateMagicBytes + StripMetadata, so DecodeConfig should succeed.
	// If it doesn't, log so an operator can notice rather than swallowing.
	if h.primer != nil {
		if cfg, _, err := image.DecodeConfig(bytes.NewReader(strippedBytes)); err == nil {
			h.primer.Prime(uploadURL, cfg.Width, cfg.Height)
		} else {
			log.Printf("upload dimensions decode failed for %s (%s): %v", uploadURL, contentType, err)
		}
	}

	// Return response
	response := ImageUploadResponse{
		URL:      uploadURL,
		Filename: sanitizeFilename(header.Filename),
	}

	writeJSON(w, http.StatusCreated, response)
}

// validateMagicBytes validates the magic bytes of a file
func validateMagicBytes(data []byte, contentType string) bool {
	expectedMagic, ok := allowedMagicBytes[contentType]
	if !ok {
		return false
	}

	if len(data) < len(expectedMagic) {
		return false
	}

	// Special validation for WebP (RIFF header + WEBP format)
	if contentType == "image/webp" {
		if len(data) < 12 {
			return false
		}
		// Check RIFF....WEBP pattern
		return bytes.HasPrefix(data, expectedMagic) && string(data[8:12]) == "WEBP"
	}

	return bytes.HasPrefix(data, expectedMagic)
}

// sanitizeFilename removes dangerous characters from a filename
func sanitizeFilename(filename string) string {
	// Remove path separators
	filename = filepath.Base(filename)
	// Remove control characters and special characters
	filename = strings.Map(func(r rune) rune {
		if r < 32 || r == '<' || r == '>' || r == ':' || r == '"' || r == '/' || r == '\\' || r == '|' || r == '?' || r == '*' {
			return -1
		}
		return r
	}, filename)
	return filename
}

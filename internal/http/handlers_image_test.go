package http

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chai2010/webp"
)

func TestNewImageHandlers(t *testing.T) {
	handlers := NewImageHandlers("/tmp/uploads", 5*1024*1024)

	if handlers.uploadDir != "/tmp/uploads" {
		t.Errorf("expected uploadDir to be /tmp/uploads, got %s", handlers.uploadDir)
	}
	if handlers.maxUploadSize != 5*1024*1024 {
		t.Errorf("expected maxUploadSize to be 5MB, got %d", handlers.maxUploadSize)
	}
}

func TestHandleUploadImage(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "upload_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	handlers := NewImageHandlers(tempDir, 5*1024*1024)

	tests := []struct {
		name           string
		setupRequest   func() (*http.Request, error)
		expectedStatus int
		expectedError  string
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "valid JPEG image upload",
			setupRequest: func() (*http.Request, error) {
				return createMultipartRequest("image", "test.jpg", createJPEGData(), "image/jpeg")
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response ImageUploadResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if !strings.HasPrefix(response.URL, "/uploads/") {
					t.Errorf("expected URL to start with /uploads/, got %s", response.URL)
				}
				if !strings.HasSuffix(response.URL, ".jpg") {
					t.Errorf("expected URL to end with .jpg, got %s", response.URL)
				}
				if response.Filename != "test.jpg" {
					t.Errorf("expected filename to be test.jpg, got %s", response.Filename)
				}
			},
		},
		{
			name: "valid PNG image upload",
			setupRequest: func() (*http.Request, error) {
				return createMultipartRequest("image", "test.png", createPNGData(), "image/png")
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response ImageUploadResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if !strings.HasSuffix(response.URL, ".png") {
					t.Errorf("expected URL to end with .png, got %s", response.URL)
				}
			},
		},
		{
			name: "valid GIF image upload",
			setupRequest: func() (*http.Request, error) {
				return createMultipartRequest("image", "test.gif", createGIFData(), "image/gif")
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response ImageUploadResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if !strings.HasSuffix(response.URL, ".gif") {
					t.Errorf("expected URL to end with .gif, got %s", response.URL)
				}
			},
		},
		{
			name: "valid WebP image upload",
			setupRequest: func() (*http.Request, error) {
				return createMultipartRequest("image", "test.webp", createWebPData(), "image/webp")
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response ImageUploadResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if !strings.HasSuffix(response.URL, ".webp") {
					t.Errorf("expected URL to end with .webp, got %s", response.URL)
				}
			},
		},
		{
			name: "no file provided",
			setupRequest: func() (*http.Request, error) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				writer.Close()
				req := httptest.NewRequest("POST", "/api/v1/images", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req, nil
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "No image file provided",
		},
		{
			name: "invalid content type",
			setupRequest: func() (*http.Request, error) {
				return createMultipartRequest("image", "test.txt", []byte("plain text"), "text/plain")
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid file type",
		},
		{
			name: "content type mismatch (fake JPEG header)",
			setupRequest: func() (*http.Request, error) {
				// Has PNG header but sent as JPEG
				return createMultipartRequest("image", "test.jpg", createPNGData(), "image/jpeg")
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid file content",
		},
		{
			name: "content type mismatch (fake PNG header)",
			setupRequest: func() (*http.Request, error) {
				// Has JPEG header but sent as PNG
				return createMultipartRequest("image", "test.png", createJPEGData(), "image/png")
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid file content",
		},
		{
			name: "wrong field name",
			setupRequest: func() (*http.Request, error) {
				return createMultipartRequest("file", "test.jpg", createJPEGData(), "image/jpeg")
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "No image file provided",
		},
		{
			name: "filename with path traversal attempt",
			setupRequest: func() (*http.Request, error) {
				return createMultipartRequest("image", "../../../etc/passwd.jpg", createJPEGData(), "image/jpeg")
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response ImageUploadResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				// Verify path traversal is neutralized
				if strings.Contains(response.Filename, "..") {
					t.Errorf("filename should not contain path traversal: %s", response.Filename)
				}
				if strings.Contains(response.URL, "..") {
					t.Errorf("URL should not contain path traversal: %s", response.URL)
				}
			},
		},
		{
			name: "filename with special characters",
			setupRequest: func() (*http.Request, error) {
				return createMultipartRequest("image", "<script>alert('xss')</script>.jpg", createJPEGData(), "image/jpeg")
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response ImageUploadResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				// Verify dangerous characters are removed
				if strings.Contains(response.Filename, "<") || strings.Contains(response.Filename, ">") {
					t.Errorf("filename should not contain special characters: %s", response.Filename)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := tt.setupRequest()
			if err != nil {
				t.Fatalf("failed to setup request: %v", err)
			}

			rr := httptest.NewRecorder()
			handlers.HandleUploadImage(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d, body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}

			if tt.expectedError != "" {
				if !strings.Contains(rr.Body.String(), tt.expectedError) {
					t.Errorf("expected error containing %q, got %q", tt.expectedError, rr.Body.String())
				}
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}
}

func TestHandleUploadImage_FileTooLarge(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "upload_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set 100 byte limit
	handlers := NewImageHandlers(tempDir, 100)

	// Create 200 byte file
	largeData := make([]byte, 200)
	copy(largeData, createJPEGData())

	req, err := createMultipartRequest("image", "large.jpg", largeData, "image/jpeg")
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handlers.HandleUploadImage(rr, req)

	// Expect size limit error
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleUploadImage_FileActuallySaved(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "upload_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	handlers := NewImageHandlers(tempDir, 5*1024*1024)

	jpegData := createJPEGData()
	req, err := createMultipartRequest("image", "test.jpg", jpegData, "image/jpeg")
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handlers.HandleUploadImage(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}

	var response ImageUploadResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify the file was actually saved
	filename := strings.TrimPrefix(response.URL, "/uploads/")
	savedPath := filepath.Join(tempDir, filename)

	savedData, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	// Verify the saved file is a valid JPEG
	// (May differ from original byte sequence due to metadata removal)
	if !bytes.HasPrefix(savedData, []byte{0xFF, 0xD8, 0xFF}) {
		t.Error("saved file is not a valid JPEG")
	}

	// Verify it can be decoded as an image
	_, err = jpeg.Decode(bytes.NewReader(savedData))
	if err != nil {
		t.Errorf("saved file cannot be decoded as JPEG: %v", err)
	}
}

func TestValidateMagicBytes(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		contentType string
		expected    bool
	}{
		{
			name:        "valid JPEG",
			data:        createJPEGData(),
			contentType: "image/jpeg",
			expected:    true,
		},
		{
			name:        "valid PNG",
			data:        createPNGData(),
			contentType: "image/png",
			expected:    true,
		},
		{
			name:        "valid GIF",
			data:        createGIFData(),
			contentType: "image/gif",
			expected:    true,
		},
		{
			name:        "valid WebP",
			data:        createWebPData(),
			contentType: "image/webp",
			expected:    true,
		},
		{
			name:        "invalid - JPEG header with PNG type",
			data:        createJPEGData(),
			contentType: "image/png",
			expected:    false,
		},
		{
			name:        "invalid - PNG header with JPEG type",
			data:        createPNGData(),
			contentType: "image/jpeg",
			expected:    false,
		},
		{
			name:        "invalid - random data",
			data:        []byte{0x00, 0x01, 0x02, 0x03},
			contentType: "image/jpeg",
			expected:    false,
		},
		{
			name:        "invalid - empty data",
			data:        []byte{},
			contentType: "image/jpeg",
			expected:    false,
		},
		{
			name:        "invalid - unsupported type",
			data:        []byte{0x00, 0x01, 0x02, 0x03},
			contentType: "image/bmp",
			expected:    false,
		},
		{
			name:        "invalid WebP - missing WEBP signature",
			data:        []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x41, 0x56, 0x49, 0x20},
			contentType: "image/webp",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateMagicBytes(tt.data, tt.contentType)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal filename",
			input:    "test.jpg",
			expected: "test.jpg",
		},
		{
			name:     "path traversal",
			input:    "../../../etc/passwd",
			expected: "passwd",
		},
		{
			name:     "windows path",
			input:    "C:\\Users\\test\\image.jpg",
			expected: "CUserstestimage.jpg", // On Unix, backslash is not a path separator, so the entire string is treated as filename with : and \ removed
		},
		{
			name:     "special characters",
			input:    "<script>alert('xss')</script>.jpg",
			expected: "script.jpg", // / is treated as path separator resulting in script>.jpg, then > is removed
		},
		{
			name:     "control characters",
			input:    "test\x00\x1f.jpg",
			expected: "test.jpg",
		},
		{
			name:     "pipe and question mark",
			input:    "file|name?.jpg",
			expected: "filename.jpg",
		},
		{
			name:     "quotes",
			input:    "file\"name.jpg",
			expected: "filename.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Helper functions

func createMultipartRequest(fieldName, filename string, data []byte, contentType string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="` + fieldName + `"; filename="` + filename + `"`}
	h["Content-Type"] = []string{contentType}

	part, err := writer.CreatePart(h)
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(part, bytes.NewReader(data)); err != nil {
		return nil, err
	}

	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/images", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, nil
}

// Helper functions to generate valid image data

func createJPEGData() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	var buf bytes.Buffer
	jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	return buf.Bytes()
}

func createPNGData() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func createGIFData() []byte {
	img := image.NewPaletted(image.Rect(0, 0, 10, 10), color.Palette{
		color.RGBA{R: 0, G: 0, B: 255, A: 255},
		color.RGBA{R: 255, G: 255, B: 255, A: 255},
	})

	var buf bytes.Buffer
	gif.Encode(&buf, img, nil)
	return buf.Bytes()
}

func createWebPData() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 255, B: 0, A: 255})
		}
	}

	var buf bytes.Buffer
	webp.Encode(&buf, img, &webp.Options{Lossless: true})
	return buf.Bytes()
}

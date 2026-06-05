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
	"sync/atomic"
	"testing"
	"time"

	"github.com/chai2010/webp"
)

func TestNewImageHandlers(t *testing.T) {
	handlers := NewImageHandlers("/tmp/uploads", 5*1024*1024, nil)

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

	handlers := NewImageHandlers(tempDir, 5*1024*1024, nil)

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
	handlers := NewImageHandlers(tempDir, 100, nil)

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

	handlers := NewImageHandlers(tempDir, 5*1024*1024, nil)

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

// recordingPrimer captures Prime calls for assertion.
type recordingPrimer struct {
	calls []primeCall
}

type primeCall struct {
	url           string
	width, height int
}

func (r *recordingPrimer) Prime(url string, width, height int) {
	r.calls = append(r.calls, primeCall{url, width, height})
}

// After a successful upload the handler must Prime the dimensions cache
// with the saved URL and the actual pixel size of the file, so the next
// public-page render doesn't have to hit disk for the same image.
func TestHandleUploadImage_PrimesDimensionsCache(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "upload_primer_test")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	primer := &recordingPrimer{}
	handlers := NewImageHandlers(tempDir, 5*1024*1024, primer)

	req, err := createMultipartRequest("image", "test.jpg", createJPEGData(), "image/jpeg")
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	rec := httptest.NewRecorder()
	handlers.HandleUploadImage(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("upload failed: %d, body=%s", rec.Code, rec.Body.String())
	}
	if len(primer.calls) != 1 {
		t.Fatalf("expected exactly 1 Prime call, got %d (%+v)", len(primer.calls), primer.calls)
	}
	c := primer.calls[0]

	// createJPEGData encodes a 10×10 image.
	if c.width != 10 || c.height != 10 {
		t.Errorf("Prime dimensions = (%d, %d), want (10, 10)", c.width, c.height)
	}

	var response ImageUploadResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if c.url != response.URL {
		t.Errorf("Prime URL = %q, want response URL %q", c.url, response.URL)
	}
}

// TestHandleUploadImage_VariantGoroutineRecoversFromPanic guards two
// pieces of wiring at once: (1) variantFn runs in a background goroutine
// the handler can survive losing, and (2) a deliberate panic inside that
// goroutine is recovered, reported via variantDone, and does NOT take
// down the process. Without recover() a malicious image whose decoder
// panics would crash the whole server.
func TestHandleUploadImage_VariantGoroutineRecoversFromPanic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "upload_variant_panic_test")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	done := make(chan error, 1)
	handlers := NewImageHandlers(tempDir, 5*1024*1024, nil)
	handlers.variantFn = func(path string) error {
		panic("simulated decoder panic")
	}
	handlers.variantDone = func(_ string, err error) {
		done <- err
	}

	req, err := createMultipartRequest("image", "test.jpg", createJPEGData(), "image/jpeg")
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	rec := httptest.NewRecorder()
	handlers.HandleUploadImage(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("handler must not surface variant failures to the client, got %d", rec.Code)
	}

	select {
	case got := <-done:
		if got == nil {
			t.Errorf("variantDone should report the recovered panic as an error, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("variantDone never fired — goroutine likely deadlocked or never started")
	}
}

// TestHandleUploadImage_VariantSemaphoreBoundsConcurrency confirms the
// per-goroutine semaphore is wired in: with capacity capped to 1, two
// uploads that block inside variantFn can't both be in flight at the
// same time.
func TestHandleUploadImage_VariantSemaphoreBoundsConcurrency(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "upload_variant_sem_test")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	handlers := NewImageHandlers(tempDir, 5*1024*1024, nil)
	// Force serialized execution by replacing the semaphore with a
	// capacity-1 channel.
	handlers.variantSem = make(chan struct{}, 1)

	var inFlight int32
	var peak int32
	release := make(chan struct{})

	handlers.variantFn = func(path string) error {
		n := atomic.AddInt32(&inFlight, 1)
		if n > atomic.LoadInt32(&peak) {
			atomic.StoreInt32(&peak, n)
		}
		<-release
		atomic.AddInt32(&inFlight, -1)
		return nil
	}
	doneCh := make(chan struct{}, 2)
	handlers.variantDone = func(_ string, _ error) {
		doneCh <- struct{}{}
	}

	for i := 0; i < 2; i++ {
		req, err := createMultipartRequest("image", "test.jpg", createJPEGData(), "image/jpeg")
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
		rec := httptest.NewRecorder()
		handlers.HandleUploadImage(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("upload %d failed: %d", i, rec.Code)
		}
	}

	// Give the second goroutine a moment to reach the semaphore wait.
	time.Sleep(50 * time.Millisecond)

	if got := atomic.LoadInt32(&peak); got > 1 {
		t.Errorf("variant goroutines ran in parallel despite capacity=1; peak in-flight = %d", got)
	}

	close(release)
	for i := 0; i < 2; i++ {
		select {
		case <-doneCh:
		case <-time.After(2 * time.Second):
			t.Fatalf("variantDone never fired for upload %d", i)
		}
	}
}

// When primer is nil the handler must still succeed (back-compat path).
func TestHandleUploadImage_NilPrimerIsNoOp(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "upload_nil_primer_test")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	handlers := NewImageHandlers(tempDir, 5*1024*1024, nil)
	req, err := createMultipartRequest("image", "test.png", createPNGData(), "image/png")
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	rec := httptest.NewRecorder()
	handlers.HandleUploadImage(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("upload with nil primer failed: %d", rec.Code)
	}
}

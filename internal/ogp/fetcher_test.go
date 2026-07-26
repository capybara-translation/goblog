package ogp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		// Loopback addresses
		{"IPv4 loopback", "127.0.0.1", true},
		{"IPv6 loopback", "::1", true},

		// Private ranges (RFC 1918)
		{"10.0.0.0/8", "10.0.0.1", true},
		{"172.16.0.0/12", "172.16.0.1", true},
		{"192.168.0.0/16", "192.168.1.1", true},

		// Link-local
		{"IPv4 link-local", "169.254.1.1", true},
		{"IPv6 link-local", "fe80::1", true},

		// Public IPs
		{"Google DNS", "8.8.8.8", false},
		{"Cloudflare DNS", "1.1.1.1", false},
		{"Public IPv6", "2001:4860:4860::8888", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}
			result := isPrivateIP(ip)
			if result != tt.expected {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestValidateHost(t *testing.T) {
	tests := []struct {
		name        string
		host        string
		expectError bool
	}{
		// Blocked hosts
		{"localhost", "localhost", true},
		{"localhost with port", "localhost:8080", true},
		{"127.0.0.1", "127.0.0.1", true},
		{"127.0.0.1 with port", "127.0.0.1:3000", true},
		{"IPv6 loopback", "::1", true},

		// Public hosts (should pass)
		{"example.com", "example.com", false},
		{"example.com with port", "example.com:443", false},

		// Non-existent domain (DNS lookup fails, but should pass)
		{"non-existent domain", "this-domain-does-not-exist-12345.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHost(tt.host)
			if tt.expectError && err == nil {
				t.Errorf("validateHost(%q) should return error", tt.host)
			}
			if !tt.expectError && err != nil {
				t.Errorf("validateHost(%q) unexpected error: %v", tt.host, err)
			}
		})
	}
}

func TestIsAmazonProductPage(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		// Amazon product pages
		{"amazon.co.jp /dp/", "https://www.amazon.co.jp/dp/B0GHLLBXSS", true},
		{"amazon.com /dp/", "https://www.amazon.com/dp/B0GHLLBXSS", true},
		{"amazon.de /dp/", "https://www.amazon.de/dp/B0GHLLBXSS", true},
		{"amazon /gp/product/", "https://www.amazon.co.jp/gp/product/B0GHLLBXSS", true},

		// Non-Amazon pages
		{"example.com", "https://example.com/dp/something", false},
		{"amazon search", "https://www.amazon.co.jp/s?k=test", false},
		{"amazon home", "https://www.amazon.co.jp/", false},

		// Edge cases
		{"amazon without product path", "https://www.amazon.co.jp/about", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAmazonProductPage(tt.url)
			if result != tt.expected {
				t.Errorf("isAmazonProductPage(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestIsAmazonProductImage(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		expected bool
	}{
		{"Product image with /I/ _SX _SY", "https://m.media-amazon.com/images/I/51abc_SX342_SY445_.jpg", true},
		{"Missing _SY", "https://m.media-amazon.com/images/I/51abc_SX342_.jpg", false},
		{"Missing _SX", "https://m.media-amazon.com/images/I/51abc_SY445_.jpg", false},
		{"Missing /I/", "https://m.media-amazon.com/images/G/logo_SX100_SY50_.jpg", false},
		{"Store logo", "https://m.media-amazon.com/images/G/09/logo.jpg", false},
		{"Empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAmazonProductImage(tt.src)
			if result != tt.expected {
				t.Errorf("isAmazonProductImage(%q) = %v, want %v", tt.src, result, tt.expected)
			}
		})
	}
}

func TestParseOGP(t *testing.T) {
	tests := []struct {
		name      string
		html      string
		url       string
		wantTitle string
		wantDesc  string
		wantImage string
		wantSite  string
	}{
		{
			name: "Full OGP tags",
			html: `<!DOCTYPE html>
<html>
<head>
	<meta property="og:title" content="Test Title">
	<meta property="og:description" content="Test Description">
	<meta property="og:image" content="https://example.com/image.jpg">
	<meta property="og:site_name" content="Test Site">
</head>
<body></body>
</html>`,
			url:       "https://example.com/page",
			wantTitle: "Test Title",
			wantDesc:  "Test Description",
			wantImage: "https://example.com/image.jpg",
			wantSite:  "Test Site",
		},
		{
			name: "Fallback to title tag",
			html: `<!DOCTYPE html>
<html>
<head>
	<title>Page Title</title>
</head>
<body></body>
</html>`,
			url:       "https://example.com/page",
			wantTitle: "Page Title",
			wantSite:  "example.com",
		},
		{
			name: "Fallback to meta description",
			html: `<!DOCTYPE html>
<html>
<head>
	<meta name="description" content="Meta Description">
	<title>Title</title>
</head>
<body></body>
</html>`,
			url:       "https://example.com/page",
			wantTitle: "Title",
			wantDesc:  "Meta Description",
			wantSite:  "example.com",
		},
		{
			name: "Twitter Card fallback",
			html: `<!DOCTYPE html>
<html>
<head>
	<meta name="twitter:title" content="Twitter Title">
	<meta name="twitter:description" content="Twitter Description">
	<meta name="twitter:image" content="https://example.com/twitter.jpg">
</head>
<body></body>
</html>`,
			url:       "https://example.com/page",
			wantTitle: "Twitter Title",
			wantDesc:  "Twitter Description",
			wantImage: "https://example.com/twitter.jpg",
			wantSite:  "example.com",
		},
		{
			name: "OGP takes precedence over Twitter Card",
			html: `<!DOCTYPE html>
<html>
<head>
	<meta property="og:title" content="OGP Title">
	<meta name="twitter:title" content="Twitter Title">
</head>
<body></body>
</html>`,
			url:       "https://example.com/page",
			wantTitle: "OGP Title",
			wantSite:  "example.com",
		},
		{
			name: "Amazon product page prefers body image over og:image",
			html: `<!DOCTYPE html>
<html>
<head>
	<meta property="og:title" content="Product Title">
	<meta property="og:image" content="https://m.media-amazon.com/images/G/store-logo.jpg">
</head>
<body>
	<img src="https://m.media-amazon.com/images/I/51product_SX342_SY445_.jpg">
</body>
</html>`,
			url:       "https://www.amazon.co.jp/dp/1234567890",
			wantTitle: "Product Title",
			wantImage: "https://m.media-amazon.com/images/I/51product_SX342_SY445_.jpg",
			wantSite:  "www.amazon.co.jp",
		},
		{
			name: "Amazon product page falls back to og:image when no body image",
			html: `<!DOCTYPE html>
<html>
<head>
	<meta property="og:title" content="Product Title">
	<meta property="og:image" content="https://m.media-amazon.com/images/G/fallback.jpg">
</head>
<body>
	<p>No product images here</p>
</body>
</html>`,
			url:       "https://www.amazon.co.jp/dp/1234567890",
			wantTitle: "Product Title",
			wantImage: "https://m.media-amazon.com/images/G/fallback.jpg",
			wantSite:  "www.amazon.co.jp",
		},
		{
			name: "Amazon ignores images without _SX and _SY",
			html: `<!DOCTYPE html>
<html>
<head>
	<meta property="og:title" content="Product Title">
	<meta property="og:image" content="https://m.media-amazon.com/images/G/og-logo.jpg">
</head>
<body>
	<img src="https://m.media-amazon.com/images/I/51abc_SL1500_.jpg">
	<img src="https://m.media-amazon.com/images/I/51product_SX342_SY445_.jpg">
</body>
</html>`,
			url:       "https://www.amazon.co.jp/dp/1234567890",
			wantTitle: "Product Title",
			wantImage: "https://m.media-amazon.com/images/I/51product_SX342_SY445_.jpg",
			wantSite:  "www.amazon.co.jp",
		},
		{
			name: "URL as title fallback",
			html: `<!DOCTYPE html>
<html>
<head></head>
<body></body>
</html>`,
			url:       "https://example.com/page",
			wantTitle: "https://example.com/page",
			wantSite:  "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := parseOGP(strings.NewReader(tt.html), tt.url)
			if err != nil {
				t.Fatalf("parseOGP() error = %v", err)
			}

			if data.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", data.Title, tt.wantTitle)
			}
			if data.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", data.Description, tt.wantDesc)
			}
			if data.ImageURL != tt.wantImage {
				t.Errorf("ImageURL = %q, want %q", data.ImageURL, tt.wantImage)
			}
			if tt.wantSite != "" && data.SiteName != tt.wantSite {
				t.Errorf("SiteName = %q, want %q", data.SiteName, tt.wantSite)
			}
		})
	}
}

// mockRoundTripper is a mock http.RoundTripper for testing
type mockRoundTripper struct {
	response *http.Response
	err      error
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

// newTestFetcher creates a fetcher with a mock HTTP client for testing
func newTestFetcher(rt http.RoundTripper) *httpFetcher {
	return &httpFetcher{
		client: &http.Client{
			Transport: rt,
			Timeout:   5 * time.Second,
		},
	}
}

func TestFetch_Success(t *testing.T) {
	htmlBody := `<!DOCTYPE html>
<html>
<head>
	<meta property="og:title" content="Test Page">
	<meta property="og:description" content="Test Description">
	<meta property="og:image" content="https://example.com/image.jpg">
</head>
<body></body>
</html>`

	mockRT := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(htmlBody)),
			Header:     make(http.Header),
		},
	}

	fetcher := newTestFetcher(mockRT)
	ctx := context.Background()

	data, err := fetcher.Fetch(ctx, "https://example.com/page")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if data.Title != "Test Page" {
		t.Errorf("Title = %q, want %q", data.Title, "Test Page")
	}
	if data.Description != "Test Description" {
		t.Errorf("Description = %q, want %q", data.Description, "Test Description")
	}
	if data.ImageURL != "https://example.com/image.jpg" {
		t.Errorf("ImageURL = %q, want %q", data.ImageURL, "https://example.com/image.jpg")
	}
}

func TestFetch_InvalidScheme(t *testing.T) {
	fetcher := NewFetcher(5 * time.Second)
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx, "ftp://example.com")
	if err == nil {
		t.Error("Fetch() should return error for ftp scheme")
	}
	if !strings.Contains(err.Error(), "unsupported scheme") {
		t.Errorf("Fetch() error = %v, want error containing 'unsupported scheme'", err)
	}
}

func TestFetch_PrivateIP(t *testing.T) {
	fetcher := NewFetcher(5 * time.Second)
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx, "http://localhost/test")
	if err == nil {
		t.Error("Fetch() should return error for localhost")
	}
	if err != ErrPrivateIP {
		t.Errorf("Fetch() error = %v, want ErrPrivateIP", err)
	}
}

func TestFetch_Non200Status(t *testing.T) {
	mockRT := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(bytes.NewBufferString("")),
			Header:     make(http.Header),
		},
	}

	fetcher := newTestFetcher(mockRT)
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx, "https://example.com/notfound")
	if err == nil {
		t.Error("Fetch() should return error for 404 status")
	}
	if !strings.Contains(err.Error(), "status code 404") {
		t.Errorf("Fetch() error = %v, want error containing 'status code 404'", err)
	}
}

func TestFetch_ContextCancellation(t *testing.T) {
	mockRT := &mockRoundTripper{
		err: context.Canceled,
	}

	fetcher := newTestFetcher(mockRT)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := fetcher.Fetch(ctx, "https://example.com/page")
	if err == nil {
		t.Error("Fetch() should return error for cancelled context")
	}
}

func TestNewFetcher(t *testing.T) {
	fetcher := NewFetcher(10 * time.Second)
	if fetcher == nil {
		t.Error("NewFetcher() returned nil")
	}
}

func TestDownloadImage_Success(t *testing.T) {
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0} // Minimal JPEG header bytes
	header := make(http.Header)
	header.Set("Content-Type", "image/jpeg")

	mockRT := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(imageData)),
			Header:     header,
		},
	}

	fetcher := newTestFetcher(mockRT)
	destDir := t.TempDir()
	ctx := context.Background()

	relativePath, err := fetcher.DownloadImage(ctx, "https://example.com/image.jpg", destDir)
	if err != nil {
		t.Fatalf("DownloadImage() error = %v", err)
	}

	// Verify relative path format: "ogp-cache/<uuid>.jpg"
	if !strings.HasPrefix(relativePath, "ogp-cache/") {
		t.Errorf("relativePath = %q, want prefix 'ogp-cache/'", relativePath)
	}
	if !strings.HasSuffix(relativePath, ".jpg") {
		t.Errorf("relativePath = %q, want suffix '.jpg'", relativePath)
	}

	// Verify file exists on disk
	fullPath := destDir + "/" + relativePath
	info, err := os.Stat(fullPath)
	if err != nil {
		t.Fatalf("saved file not found: %v", err)
	}
	if info.Size() != int64(len(imageData)) {
		t.Errorf("file size = %d, want %d", info.Size(), len(imageData))
	}
}

func TestDownloadImage_SuccessPNG(t *testing.T) {
	imageData := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic bytes
	header := make(http.Header)
	header.Set("Content-Type", "image/png")

	mockRT := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(imageData)),
			Header:     header,
		},
	}

	fetcher := newTestFetcher(mockRT)
	destDir := t.TempDir()
	ctx := context.Background()

	relativePath, err := fetcher.DownloadImage(ctx, "https://example.com/image.png", destDir)
	if err != nil {
		t.Fatalf("DownloadImage() error = %v", err)
	}

	if !strings.HasSuffix(relativePath, ".png") {
		t.Errorf("relativePath = %q, want suffix '.png'", relativePath)
	}
}

func TestDownloadImage_ContentTypeWithCharset(t *testing.T) {
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	header := make(http.Header)
	header.Set("Content-Type", "image/jpeg; charset=utf-8")

	mockRT := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(imageData)),
			Header:     header,
		},
	}

	fetcher := newTestFetcher(mockRT)
	destDir := t.TempDir()
	ctx := context.Background()

	relativePath, err := fetcher.DownloadImage(ctx, "https://example.com/image.jpg", destDir)
	if err != nil {
		t.Fatalf("DownloadImage() error = %v", err)
	}

	if !strings.HasSuffix(relativePath, ".jpg") {
		t.Errorf("relativePath = %q, want suffix '.jpg'", relativePath)
	}
}

func TestDownloadImage_TooLarge(t *testing.T) {
	// Create data larger than maxImageSize (2MB)
	largeData := make([]byte, maxImageSize+100)
	header := make(http.Header)
	header.Set("Content-Type", "image/jpeg")

	mockRT := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(largeData)),
			Header:     header,
		},
	}

	fetcher := newTestFetcher(mockRT)
	destDir := t.TempDir()
	ctx := context.Background()

	_, err := fetcher.DownloadImage(ctx, "https://example.com/huge.jpg", destDir)
	if err == nil {
		t.Fatal("DownloadImage() should return error for oversized image")
	}
	if !strings.Contains(err.Error(), "image too large") {
		t.Errorf("DownloadImage() error = %v, want error containing 'image too large'", err)
	}
}

func TestDownloadImage_InvalidContentType(t *testing.T) {
	header := make(http.Header)
	header.Set("Content-Type", "text/html")

	mockRT := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("<html></html>")),
			Header:     header,
		},
	}

	fetcher := newTestFetcher(mockRT)
	destDir := t.TempDir()
	ctx := context.Background()

	_, err := fetcher.DownloadImage(ctx, "https://example.com/notimage", destDir)
	if err == nil {
		t.Fatal("DownloadImage() should return error for non-image Content-Type")
	}
	if !strings.Contains(err.Error(), "unsupported content type") {
		t.Errorf("DownloadImage() error = %v, want error containing 'unsupported content type'", err)
	}
}

func TestDownloadImage_SSRFPrevention(t *testing.T) {
	fetcher := NewFetcher(5 * time.Second)
	ctx := context.Background()
	destDir := t.TempDir()

	tests := []struct {
		name string
		url  string
	}{
		{"localhost", "http://localhost/image.jpg"},
		{"127.0.0.1", "http://127.0.0.1/image.jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := fetcher.DownloadImage(ctx, tt.url, destDir)
			if err == nil {
				t.Errorf("DownloadImage(%q) should return error for private IP", tt.url)
			}
			if !errors.Is(err, ErrPrivateIP) {
				t.Errorf("DownloadImage(%q) error = %v, want ErrPrivateIP", tt.url, err)
			}
		})
	}
}

func TestDownloadImage_InvalidScheme(t *testing.T) {
	fetcher := NewFetcher(5 * time.Second)
	ctx := context.Background()
	destDir := t.TempDir()

	_, err := fetcher.DownloadImage(ctx, "ftp://example.com/image.jpg", destDir)
	if err == nil {
		t.Fatal("DownloadImage() should return error for ftp scheme")
	}
	if !strings.Contains(err.Error(), "unsupported scheme") {
		t.Errorf("DownloadImage() error = %v, want error containing 'unsupported scheme'", err)
	}
}

func TestDownloadImage_Non200Status(t *testing.T) {
	mockRT := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusForbidden,
			Body:       io.NopCloser(bytes.NewBufferString("")),
			Header:     make(http.Header),
		},
	}

	fetcher := newTestFetcher(mockRT)
	destDir := t.TempDir()
	ctx := context.Background()

	_, err := fetcher.DownloadImage(ctx, "https://example.com/forbidden.jpg", destDir)
	if err == nil {
		t.Fatal("DownloadImage() should return error for 403 status")
	}
	if !strings.Contains(err.Error(), "status code 403") {
		t.Errorf("DownloadImage() error = %v, want error containing 'status code 403'", err)
	}
}

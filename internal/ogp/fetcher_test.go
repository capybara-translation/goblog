package ogp

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
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

func TestSelectBestAmazonImage(t *testing.T) {
	tests := []struct {
		name       string
		candidates []string
		expected   string
	}{
		{
			name:       "Empty candidates",
			candidates: []string{},
			expected:   "",
		},
		{
			name: "Priority 1: _SL1500_",
			candidates: []string{
				"https://m.media-amazon.com/images/I/abc_SX50_.jpg",
				"https://m.media-amazon.com/images/I/abc_SL1500_.jpg",
				"https://m.media-amazon.com/images/I/abc_AC_.jpg",
			},
			expected: "https://m.media-amazon.com/images/I/abc_SL1500_.jpg",
		},
		{
			name: "Priority 1: _SL1080_",
			candidates: []string{
				"https://m.media-amazon.com/images/I/abc_SX50_.jpg",
				"https://m.media-amazon.com/images/I/abc_SL1080_.jpg",
			},
			expected: "https://m.media-amazon.com/images/I/abc_SL1080_.jpg",
		},
		{
			name: "Priority 2: _SX679_",
			candidates: []string{
				"https://m.media-amazon.com/images/I/abc_SS75_.jpg",
				"https://m.media-amazon.com/images/I/abc_SX679_.jpg",
			},
			expected: "https://m.media-amazon.com/images/I/abc_SX679_.jpg",
		},
		{
			name: "Priority 3: _AC_ (not thumbnail)",
			candidates: []string{
				"https://m.media-amazon.com/images/I/abc_SS75_.jpg",
				"https://m.media-amazon.com/images/I/abc_AC_SX300_.jpg",
			},
			expected: "https://m.media-amazon.com/images/I/abc_AC_SX300_.jpg",
		},
		{
			name: "Priority 3: Skip _AC_ with thumbnail size",
			candidates: []string{
				"https://m.media-amazon.com/images/I/abc_AC_SS75_.jpg",
				"https://m.media-amazon.com/images/I/abc_medium.jpg",
			},
			expected: "https://m.media-amazon.com/images/I/abc_medium.jpg",
		},
		{
			name: "Fallback to first candidate",
			candidates: []string{
				"https://m.media-amazon.com/images/I/abc_SS75_.jpg",
				"https://m.media-amazon.com/images/I/def_SX50_.jpg",
			},
			expected: "https://m.media-amazon.com/images/I/abc_SS75_.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selectBestAmazonImage(tt.candidates)
			if result != tt.expected {
				t.Errorf("selectBestAmazonImage() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParseOGP(t *testing.T) {
	tests := []struct {
		name        string
		html        string
		url         string
		wantTitle   string
		wantDesc    string
		wantImage   string
		wantSite    string
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

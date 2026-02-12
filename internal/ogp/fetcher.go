package ogp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Fetcher defines the interface for fetching OGP data from URLs
type Fetcher interface {
	// Fetch retrieves OGP data from the given URL
	Fetch(ctx context.Context, url string) (*Data, error)
}

// Common errors
var (
	ErrInvalidURL    = errors.New("invalid URL")
	ErrPrivateIP     = errors.New("private IP addresses are not allowed")
	ErrFetchFailed   = errors.New("failed to fetch URL")
	ErrParseFailed   = errors.New("failed to parse HTML")
	ErrTooManyRedirects = errors.New("too many redirects")
)

// Maximum response body size (1MB)
const maxBodySize = 1 * 1024 * 1024

// httpFetcher implements the Fetcher interface using HTTP
type httpFetcher struct {
	client *http.Client
}

// NewFetcher creates a new HTTP-based OGP fetcher
func NewFetcher(timeout time.Duration) Fetcher {
	// Custom transport with TLS configuration for broader compatibility
	// Some servers require specific cipher suites
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			},
		},
	}

	return &httpFetcher{
		client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return ErrTooManyRedirects
				}
				return nil
			},
		},
	}
}

// Fetch retrieves OGP data from the given URL
func (f *httpFetcher) Fetch(ctx context.Context, targetURL string) (*Data, error) {
	// Validate and parse URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	// Only allow http and https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("%w: unsupported scheme %s", ErrInvalidURL, parsedURL.Scheme)
	}

	// Check for private IP addresses (SSRF prevention)
	if err := validateHost(parsedURL.Host); err != nil {
		return nil, err
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchFailed, err)
	}

	// Set headers to mimic a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ja,en;q=0.5")

	// Execute request
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchFailed, err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status code %d", ErrFetchFailed, resp.StatusCode)
	}

	// Limit response body size
	limitedReader := io.LimitReader(resp.Body, maxBodySize)

	// Parse HTML and extract OGP data
	data, err := parseOGP(limitedReader, targetURL)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// validateHost checks if the host is safe to fetch (no private IPs)
func validateHost(host string) error {
	// Remove port if present
	hostname := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		hostname = h
	}

	// Check for localhost
	if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
		return ErrPrivateIP
	}

	// Resolve hostname and check IPs
	ips, err := net.LookupIP(hostname)
	if err != nil {
		// Allow if DNS lookup fails - let the HTTP client handle it
		return nil
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return ErrPrivateIP
		}
	}

	return nil
}

// isPrivateIP checks if an IP address is private
func isPrivateIP(ip net.IP) bool {
	// Check for loopback
	if ip.IsLoopback() {
		return true
	}

	// Check for private ranges
	if ip.IsPrivate() {
		return true
	}

	// Check for link-local
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	return false
}

// isAmazonProductPage checks if the URL is an Amazon product page
func isAmazonProductPage(targetURL string) bool {
	// Match amazon.co.jp, amazon.com, amazon.de, etc.
	return strings.Contains(targetURL, "amazon.") &&
		(strings.Contains(targetURL, "/dp/") || strings.Contains(targetURL, "/gp/product/"))
}

// parseOGP extracts OGP data from HTML
func parseOGP(r io.Reader, originalURL string) (*Data, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParseFailed, err)
	}

	data := &Data{
		URL: originalURL,
	}

	isAmazon := isAmazonProductPage(originalURL)

	// Extract domain as fallback site name
	if parsedURL, err := url.Parse(originalURL); err == nil {
		data.SiteName = parsedURL.Host
	}

	// Walk the HTML tree
	var extractMetadata func(*html.Node)
	extractMetadata = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "meta":
				handleMetaTag(n, data)
			case "title":
				// Fallback title from <title> tag
				if data.Title == "" && n.FirstChild != nil {
					data.Title = strings.TrimSpace(n.FirstChild.Data)
				}
			case "body":
				// For Amazon pages, continue searching for product images
				if isAmazon && data.ImageURL == "" {
					extractAmazonImage(n, data)
				}
				return
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractMetadata(c)
		}
	}

	extractMetadata(doc)

	// If no title found, use URL as fallback
	if data.Title == "" {
		data.Title = originalURL
	}

	return data, nil
}

// extractAmazonImage extracts product image from Amazon page body
func extractAmazonImage(body *html.Node, data *Data) {
	var candidates []string

	var findImages func(*html.Node)
	findImages = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			src := ""
			for _, attr := range n.Attr {
				if attr.Key == "src" {
					src = attr.Val
					break
				}
			}

			// Look for Amazon product images
			// Pattern: https://m.media-amazon.com/images/I/XXXXX.jpg
			if strings.Contains(src, "m.media-amazon.com/images/I/") &&
				(strings.HasSuffix(src, ".jpg") || strings.HasSuffix(src, ".png") || strings.HasSuffix(src, ".webp")) {
				candidates = append(candidates, src)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findImages(c)
		}
	}

	findImages(body)

	// Select the best image from candidates
	data.ImageURL = selectBestAmazonImage(candidates)
}

// selectBestAmazonImage selects the best product image from candidates
func selectBestAmazonImage(candidates []string) string {
	if len(candidates) == 0 {
		return ""
	}

	// Priority 1: Large images with _SL1080_ or similar (highest quality)
	for _, src := range candidates {
		if strings.Contains(src, "_SL1080_") || strings.Contains(src, "_SL1500_") {
			return src
		}
	}

	// Priority 2: Medium-large images with _SX679_ or larger
	for _, src := range candidates {
		if strings.Contains(src, "_SX679_") || strings.Contains(src, "_SX569_") ||
			strings.Contains(src, "_SX522_") || strings.Contains(src, "_SX466_") {
			return src
		}
	}

	// Priority 3: Images with _AC_ prefix (auto-cropped, usually main product)
	// but not tiny thumbnails
	for _, src := range candidates {
		if strings.Contains(src, "_AC_") &&
			!strings.Contains(src, "_SS75_") &&
			!strings.Contains(src, "_SR38,") &&
			!strings.Contains(src, "_SX38_") &&
			!strings.Contains(src, "_SY50_") {
			return src
		}
	}

	// Priority 4: Any image that's not a tiny thumbnail
	for _, src := range candidates {
		if !strings.Contains(src, "_SS75_") &&
			!strings.Contains(src, "_SR38,") &&
			!strings.Contains(src, "_SX38_") &&
			!strings.Contains(src, "_SY50_") &&
			!strings.Contains(src, "_SX50_") {
			return src
		}
	}

	// Fallback: first candidate
	return candidates[0]
}

// handleMetaTag extracts OGP and fallback metadata from meta tags
func handleMetaTag(n *html.Node, data *Data) {
	var property, name, content string

	for _, attr := range n.Attr {
		switch attr.Key {
		case "property":
			property = attr.Val
		case "name":
			name = attr.Val
		case "content":
			content = attr.Val
		}
	}

	// OGP tags (og:*)
	switch property {
	case "og:title":
		data.Title = content
	case "og:description":
		data.Description = content
	case "og:image":
		data.ImageURL = content
	case "og:site_name":
		data.SiteName = content
	}

	// Twitter Card tags as fallback
	switch name {
	case "twitter:title":
		if data.Title == "" {
			data.Title = content
		}
	case "twitter:description":
		if data.Description == "" {
			data.Description = content
		}
	case "twitter:image":
		if data.ImageURL == "" {
			data.ImageURL = content
		}
	case "description":
		// Standard meta description as fallback
		if data.Description == "" {
			data.Description = content
		}
	}
}

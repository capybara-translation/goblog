package http

import (
	"net/http/httptest"
	"testing"
)

func TestClientIP_NoTrustedProxies_IgnoresXFF(t *testing.T) {
	req := httptest.NewRequest("POST", "/", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")

	if got := clientIP(req, nil); got != "203.0.113.5" {
		t.Fatalf("expected RemoteAddr IP when no trusted proxies, got %q", got)
	}
}

func TestClientIP_UntrustedDirectPeer_IgnoresHeaders(t *testing.T) {
	trusted := parseTrustedProxies([]string{"127.0.0.1"})
	req := httptest.NewRequest("POST", "/", nil)
	req.RemoteAddr = "9.9.9.9:1234" // direct peer is NOT a trusted proxy
	req.Header.Set("X-Forwarded-For", "1.1.1.1")
	req.Header.Set("X-Real-IP", "2.2.2.2")

	if got := clientIP(req, trusted); got != "9.9.9.9" {
		t.Fatalf("untrusted peer must use RemoteAddr, got %q", got)
	}
}

// The core security property: with nginx appending $remote_addr, a client can
// only PREPEND values (leftmost). The real client is the rightmost non-trusted
// entry. A spoofed leftmost value must be ignored.
func TestClientIP_TrustedProxy_ReturnsRightmostUntrusted(t *testing.T) {
	trusted := parseTrustedProxies([]string{"127.0.0.1"})
	req := httptest.NewRequest("POST", "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	// Attacker prepended "1.1.1.1"; nginx appended the real client.
	req.Header.Set("X-Forwarded-For", "1.1.1.1, 203.0.113.50")

	if got := clientIP(req, trusted); got != "203.0.113.50" {
		t.Fatalf("expected real client 203.0.113.50 (rightmost untrusted), got %q", got)
	}
}

func TestClientIP_TrustedProxy_SingleEntry(t *testing.T) {
	trusted := parseTrustedProxies([]string{"127.0.0.1"})
	req := httptest.NewRequest("POST", "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.50")

	if got := clientIP(req, trusted); got != "203.0.113.50" {
		t.Fatalf("expected 203.0.113.50, got %q", got)
	}
}

// Multi-hop (e.g. CDN -> nginx -> app): both hops are trusted; peel them off
// from the right to reach the real client.
func TestClientIP_MultiHop_StripsTrustedChain(t *testing.T) {
	trusted := parseTrustedProxies([]string{"127.0.0.1", "198.51.100.0/24"})
	req := httptest.NewRequest("POST", "/", nil)
	req.RemoteAddr = "127.0.0.1:1234" // nginx
	// client -> CDN(198.51.100.7) -> nginx
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 198.51.100.7")

	if got := clientIP(req, trusted); got != "203.0.113.50" {
		t.Fatalf("expected real client behind CDN, got %q", got)
	}
}

// Attacker cannot impersonate by stuffing fake trusted-looking hops: nginx
// appends the attacker's real address at the very right, which wins.
func TestClientIP_SpoofedTrustedHopsIgnored(t *testing.T) {
	trusted := parseTrustedProxies([]string{"127.0.0.1", "198.51.100.0/24"})
	req := httptest.NewRequest("POST", "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	// Attacker (real 9.9.9.9) sent "203.0.113.50, 198.51.100.7"; nginx appended 9.9.9.9.
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 198.51.100.7, 9.9.9.9")

	if got := clientIP(req, trusted); got != "9.9.9.9" {
		t.Fatalf("expected the appended real peer 9.9.9.9, got %q", got)
	}
}

func TestClientIP_TrimsWhitespaceAndEmptyEntries(t *testing.T) {
	trusted := parseTrustedProxies([]string{"127.0.0.1"})
	req := httptest.NewRequest("POST", "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "1.1.1.1 ,  , 203.0.113.50 ")

	if got := clientIP(req, trusted); got != "203.0.113.50" {
		t.Fatalf("expected trimmed 203.0.113.50, got %q", got)
	}
}

func TestClientIP_IPv6(t *testing.T) {
	trusted := parseTrustedProxies([]string{"127.0.0.1"})
	req := httptest.NewRequest("POST", "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "2001:db8::1")

	if got := clientIP(req, trusted); got != "2001:db8::1" {
		t.Fatalf("expected IPv6 client, got %q", got)
	}
}

// When XFF yields nothing usable (absent, or all entries are trusted proxies),
// fall back to X-Real-IP, which nginx sets to its immediate peer.
func TestClientIP_FallsBackToXRealIP(t *testing.T) {
	trusted := parseTrustedProxies([]string{"127.0.0.1"})

	t.Run("XFF absent", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		req.Header.Set("X-Real-IP", "5.6.7.8")
		if got := clientIP(req, trusted); got != "5.6.7.8" {
			t.Fatalf("expected X-Real-IP, got %q", got)
		}
	})

	t.Run("XFF only trusted entries", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		req.Header.Set("X-Forwarded-For", "127.0.0.1")
		req.Header.Set("X-Real-IP", "5.6.7.8")
		if got := clientIP(req, trusted); got != "5.6.7.8" {
			t.Fatalf("expected fallback to X-Real-IP, got %q", got)
		}
	})
}

func TestClientIP_FallsBackToRemoteAddr_WhenNoUsableHeaders(t *testing.T) {
	trusted := parseTrustedProxies([]string{"127.0.0.1"})
	req := httptest.NewRequest("POST", "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "127.0.0.1") // all trusted, no X-Real-IP

	if got := clientIP(req, trusted); got != "127.0.0.1" {
		t.Fatalf("expected RemoteAddr fallback, got %q", got)
	}
}

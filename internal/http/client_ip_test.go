package http

import (
	"net/http/httptest"
	"testing"
)

func TestClientIP_NoTrustedProxies_IgnoresXFF(t *testing.T) {
	req := httptest.NewRequest("POST", "/", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")

	got := clientIP(req, nil)
	if got != "203.0.113.5" {
		t.Fatalf("expected RemoteAddr IP when no trusted proxies, got %q", got)
	}
}

func TestClientIP_TrustedProxy_UsesXFF(t *testing.T) {
	trusted := parseTrustedProxies([]string{"203.0.113.5"})
	req := httptest.NewRequest("POST", "/", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 10.0.0.1")

	got := clientIP(req, trusted)
	if got != "1.2.3.4" {
		t.Fatalf("expected first XFF IP from trusted proxy, got %q", got)
	}
}

func TestClientIP_TrustedProxy_XRealIP(t *testing.T) {
	trusted := parseTrustedProxies([]string{"203.0.113.5"})
	req := httptest.NewRequest("POST", "/", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	req.Header.Set("X-Real-IP", "5.6.7.8")

	got := clientIP(req, trusted)
	if got != "5.6.7.8" {
		t.Fatalf("expected X-Real-IP from trusted proxy, got %q", got)
	}
}

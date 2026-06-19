package http

import (
	"net"
	"net/http"
	"strings"
)

// clientIP extracts the real client IP, consulting forwarding headers only when
// the direct peer (RemoteAddr) is itself a trusted proxy.
//
// X-Forwarded-For is parsed RIGHT-TO-LEFT: each proxy appends the address it
// received the request from, so the rightmost entries are the ones our own
// infrastructure observed. We skip entries that are trusted proxies (the proxy
// chain) and return the first non-trusted entry — the real client at the trust
// boundary. A client can only PREPEND values (leftmost), so a spoofed
// X-Forwarded-For header can never reach this result. This is the canonical
// defense against XFF spoofing (taking the leftmost entry, as before, let any
// client forge its IP and defeat per-IP rate limiting / brute-force throttling).
//
// Mirrors AuthHandlers.getClientIP but is a free function so non-auth handlers
// (reactions) can reuse it. parseTrustedProxies and extractIP live in
// handlers_auth.go (same package).
func clientIP(r *http.Request, trusted []*net.IPNet) string {
	remoteIP := extractIP(r.RemoteAddr)

	// Only trust forwarding headers when the request actually came through a
	// trusted proxy; otherwise the peer is the client (or an unknown sender)
	// and any forwarding headers are untrustworthy.
	if !isTrustedProxyIP(remoteIP, trusted) {
		return remoteIP
	}

	// Walk X-Forwarded-For from the right, skipping our trusted proxies.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(parts[i])
			if ip == "" {
				continue
			}
			if !isTrustedProxyIP(ip, trusted) {
				return ip
			}
		}
		// Every entry was a trusted proxy — fall through to other signals.
	}

	// No usable XFF entry. X-Real-IP (nginx sets it to its immediate peer) is a
	// reasonable single-hop fallback.
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	return remoteIP
}

// isTrustedProxyIP reports whether ipStr is contained in any trusted network.
func isTrustedProxyIP(ipStr string, trusted []*net.IPNet) bool {
	if len(trusted) == 0 {
		return false
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, n := range trusted {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

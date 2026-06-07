package http

import (
	"net"
	"net/http"
	"strings"
)

// clientIP extracts the client IP, trusting X-Forwarded-For / X-Real-IP only
// when the direct peer is in trusted. Mirrors AuthHandlers.getClientIP but is a
// free function so non-auth handlers (reactions) can reuse it.
// parseTrustedProxies and extractIP live in handlers_auth.go (same package).
func clientIP(r *http.Request, trusted []*net.IPNet) string {
	remoteIP := extractIP(r.RemoteAddr)
	if isTrustedProxyIP(remoteIP, trusted) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if idx := strings.Index(xff, ","); idx != -1 {
				return strings.TrimSpace(xff[:idx])
			}
			return strings.TrimSpace(xff)
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return strings.TrimSpace(xri)
		}
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

package http

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// maxXFFHops bounds how many X-Forwarded-For entries we examine (scanning from
// the right). nginx already caps total header size, but we also guard here so a
// long forwarded chain forwarded by a trusted proxy can't turn into unbounded
// per-request work. Beyond the cap we fail closed (fall back to RemoteAddr).
const maxXFFHops = 50

// clientIP extracts the real client IP, consulting forwarding headers only when
// the direct peer (RemoteAddr) is itself a trusted proxy.
//
// X-Forwarded-For is parsed RIGHT-TO-LEFT: each proxy appends the address it
// received the request from, so the rightmost entries are what our own
// infrastructure observed. We skip entries that are trusted proxies (the proxy
// chain) and return the first non-trusted entry — the real client at the trust
// boundary. A client can only PREPEND values (leftmost), so a spoofed
// X-Forwarded-For header can never reach this result. This is the canonical
// defense against XFF spoofing (taking the leftmost entry would let any client
// forge its IP and defeat per-IP rate limiting / brute-force throttling).
//
// Each entry is parsed with net/netip and normalized: an optional port and
// IPv6 brackets are stripped, and the result is returned in canonical form so
// equivalent IPv6 spellings map to the same rate-limit key. Malformed entries
// (hostnames, "unknown", bad ports) fail closed — we fall back to RemoteAddr
// rather than trust a value we cannot parse.
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

	if ip, ok := rightmostUntrustedXFF(r.Header.Get("X-Forwarded-For"), trusted); ok {
		return ip
	}

	// No usable XFF entry. X-Real-IP (nginx sets it to its immediate peer) is a
	// reasonable single-hop fallback, but validate it the same way: ignore it
	// if it is malformed or is itself a trusted proxy.
	if addr, ok := parseForwardedAddr(r.Header.Get("X-Real-IP")); ok && !isTrustedProxyIP(addr.String(), trusted) {
		return addr.String()
	}

	return remoteIP
}

// rightmostUntrustedXFF walks X-Forwarded-For right-to-left (without allocating
// the full split), peeling off trusted proxies, and returns the first
// non-trusted address in canonical form. Returns ok=false when the header is
// empty, every scanned entry is a trusted proxy, the hop cap is hit, or a
// scanned entry is malformed — in all of which the caller falls back to
// RemoteAddr (fail closed). The common case (real client appended at the right)
// resolves in a single iteration.
func rightmostUntrustedXFF(xff string, trusted []*net.IPNet) (string, bool) {
	for hops := 0; xff != "" && hops < maxXFFHops; hops++ {
		var entry string
		if idx := strings.LastIndexByte(xff, ','); idx >= 0 {
			entry, xff = xff[idx+1:], xff[:idx]
		} else {
			entry, xff = xff, ""
		}
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		addr, ok := parseForwardedAddr(entry)
		if !ok {
			// A garbage entry within the trusted suffix means we cannot trust
			// the chain; fail closed.
			return "", false
		}
		if !isTrustedProxyIP(addr.String(), trusted) {
			return addr.String(), true
		}
	}
	return "", false
}

// parseForwardedAddr parses one forwarding-header value into a canonical IP,
// accepting "ip", "ip:port", "[ip6]", and "[ip6]:port". Returns ok=false for
// anything else (hostnames, the "unknown" token, malformed ports, empty).
func parseForwardedAddr(s string) (netip.Addr, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return netip.Addr{}, false
	}
	if ap, err := netip.ParseAddrPort(s); err == nil {
		return ap.Addr().Unmap(), true
	}
	if len(s) >= 2 && s[0] == '[' && s[len(s)-1] == ']' {
		s = s[1 : len(s)-1]
	}
	if a, err := netip.ParseAddr(s); err == nil {
		return a.Unmap(), true
	}
	return netip.Addr{}, false
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

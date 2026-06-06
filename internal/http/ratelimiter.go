package http

import (
	"sync"
	"time"
)

// maxTrackedIPs bounds the limiter map. When exceeded on insert we sweep
// expired windows. This is a single-instance, in-memory limiter; a multi-
// instance deployment would move this to a shared store (e.g. Redis).
const maxTrackedIPs = 10000

type rlWindow struct {
	count   int
	resetAt time.Time
}

// ipRateLimiter is a fixed-window per-IP rate limiter.
type ipRateLimiter struct {
	mu      sync.Mutex
	windows map[string]*rlWindow
	limit   int
	window  time.Duration
	now     func() time.Time // injectable for tests
}

func newIPRateLimiter(limit int, window time.Duration) *ipRateLimiter {
	return &ipRateLimiter{
		windows: make(map[string]*rlWindow),
		limit:   limit,
		window:  window,
		now:     time.Now,
	}
}

// Allow reports whether a request from ip is permitted, consuming one unit of
// the current window's budget when it is.
func (l *ipRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	w, ok := l.windows[ip]
	if !ok || now.After(w.resetAt) {
		if len(l.windows) > maxTrackedIPs {
			l.sweepExpired(now)
		}
		l.windows[ip] = &rlWindow{count: 1, resetAt: now.Add(l.window)}
		return true
	}
	if w.count >= l.limit {
		return false
	}
	w.count++
	return true
}

// sweepExpired removes windows whose reset time has passed. Caller holds l.mu.
func (l *ipRateLimiter) sweepExpired(now time.Time) {
	for ip, w := range l.windows {
		if now.After(w.resetAt) {
			delete(l.windows, ip)
		}
	}
}

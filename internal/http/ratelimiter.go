package http

import (
	"sync"
	"time"
)

// maxTrackedIPs is a hard cap on the limiter map size. When a new IP would push
// the map past it, we first reclaim expired windows; if the map is still full
// (a flood of many distinct, not-yet-expired IPs within one window), we refuse
// to track further new IPs so the map cannot grow without bound. Already-tracked
// IPs keep being rate-limited normally. This is a single-instance, in-memory
// limiter; a multi-instance deployment would move this to a shared store
// (e.g. Redis).
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
	if w, ok := l.windows[ip]; ok && now.Before(w.resetAt) {
		// Existing, unexpired window for this IP. The window is active while
		// now < resetAt; at exactly resetAt it is considered expired (see below).
		if w.count >= l.limit {
			return false
		}
		w.count++
		return true
	}

	// New IP, or its previous window has expired and is being replaced. Before
	// inserting, enforce the hard cap on map size to guard against a memory DoS
	// from a flood of distinct IPs within a single window.
	if len(l.windows) >= maxTrackedIPs {
		l.sweepExpired(now)
		if len(l.windows) >= maxTrackedIPs {
			// Table still full of live windows — nothing to reclaim. Refuse to
			// track this new IP rather than grow unbounded. (Already-tracked
			// IPs above are unaffected.)
			return false
		}
	}
	l.windows[ip] = &rlWindow{count: 1, resetAt: now.Add(l.window)}
	return true
}

// sweepExpired removes windows whose reset time has passed. A window is expired
// once now >= resetAt (i.e. not before it), matching the boundary used by Allow.
// Caller holds l.mu.
func (l *ipRateLimiter) sweepExpired(now time.Time) {
	for ip, w := range l.windows {
		if !now.Before(w.resetAt) {
			delete(l.windows, ip)
		}
	}
}

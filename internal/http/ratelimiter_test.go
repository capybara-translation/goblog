package http

import (
	"testing"
	"time"
)

func TestIPRateLimiter_AllowsUpToLimit(t *testing.T) {
	now := time.Unix(1000, 0)
	l := newIPRateLimiter(3, time.Minute)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if !l.Allow("1.2.3.4") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if l.Allow("1.2.3.4") {
		t.Fatal("4th request should be blocked")
	}
}

func TestIPRateLimiter_ResetsAfterWindow(t *testing.T) {
	now := time.Unix(1000, 0)
	l := newIPRateLimiter(1, time.Minute)
	l.now = func() time.Time { return now }

	if !l.Allow("1.2.3.4") {
		t.Fatal("first request should be allowed")
	}
	if l.Allow("1.2.3.4") {
		t.Fatal("second request in window should be blocked")
	}

	now = now.Add(2 * time.Minute) // advance past the window
	if !l.Allow("1.2.3.4") {
		t.Fatal("request after window reset should be allowed")
	}
}

func TestIPRateLimiter_PerIP(t *testing.T) {
	now := time.Unix(1000, 0)
	l := newIPRateLimiter(1, time.Minute)
	l.now = func() time.Time { return now }

	if !l.Allow("1.1.1.1") {
		t.Fatal("ip1 should be allowed")
	}
	if !l.Allow("2.2.2.2") {
		t.Fatal("ip2 should be allowed independently")
	}
}

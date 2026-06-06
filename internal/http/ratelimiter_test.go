package http

import (
	"fmt"
	"testing"
	"time"
)

func TestIPRateLimiter_HardCap(t *testing.T) {
	now := time.Unix(1000, 0)
	l := newIPRateLimiter(1, time.Minute)
	l.now = func() time.Time { return now }

	// Fill the table to the hard cap with distinct, unexpired IPs.
	for i := 0; i < maxTrackedIPs; i++ {
		ip := fmt.Sprintf("10.%d.%d.%d", i>>16&0xff, i>>8&0xff, i&0xff)
		if !l.Allow(ip) {
			t.Fatalf("IP #%d should be allowed while under the cap", i)
		}
	}

	// A further distinct IP cannot be tracked: the table is full of live
	// windows and nothing can be reclaimed, so it is refused rather than
	// growing the map past the cap.
	if l.Allow("203.0.113.1") {
		t.Fatal("new IP past the hard cap should be refused")
	}
	if got := len(l.windows); got > maxTrackedIPs {
		t.Fatalf("map grew past the hard cap: %d > %d", got, maxTrackedIPs)
	}

	// Once the window passes, expired entries are reclaimed and new IPs are
	// tracked again.
	now = now.Add(2 * time.Minute)
	if !l.Allow("203.0.113.2") {
		t.Fatal("new IP should be allowed after windows expire and are swept")
	}
}

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

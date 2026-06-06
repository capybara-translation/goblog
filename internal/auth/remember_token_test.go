package auth

import (
	"strings"
	"testing"
)

func TestGenerateSelector_HasExpectedShape(t *testing.T) {
	s := GenerateSelector()
	if len(s) == 0 {
		t.Fatal("selector must not be empty")
	}
	// base64.URLEncoding of 16 bytes is 24 chars (with padding).
	if len(s) != 24 {
		t.Errorf("selector length = %d, want 24", len(s))
	}
}

func TestGenerateSelector_IsRandom(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		s := GenerateSelector()
		if seen[s] {
			t.Fatalf("collision after %d iterations: %q", i, s)
		}
		seen[s] = true
	}
}

func TestEncodeDecodeRememberCookie(t *testing.T) {
	s, r := "selector-x", "raw-y"
	cookie := EncodeRememberCookie(s, r)
	if !strings.Contains(cookie, ":") {
		t.Errorf("encoded cookie must contain separator: %q", cookie)
	}

	gotSel, gotRaw, ok := DecodeRememberCookie(cookie)
	if !ok {
		t.Fatal("decode failed for round-trip value")
	}
	if gotSel != s || gotRaw != r {
		t.Errorf("decoded = (%q, %q), want (%q, %q)", gotSel, gotRaw, s, r)
	}
}

func TestDecodeRememberCookie_Malformed(t *testing.T) {
	tests := []string{"", "no-separator", ":empty-selector", "empty-raw:", "two:colons:bad"}
	for _, in := range tests {
		t.Run(in, func(t *testing.T) {
			if _, _, ok := DecodeRememberCookie(in); ok {
				t.Errorf("expected decode failure for %q", in)
			}
		})
	}
}

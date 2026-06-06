package auth

import "testing"

func TestGenerateRawToken_HasExpectedShape(t *testing.T) {
	r := GenerateRawToken()
	// base64.URLEncoding of 32 bytes is 44 chars (with padding).
	if len(r) != 44 {
		t.Errorf("raw token length = %d, want 44", len(r))
	}
}

func TestGenerateRawToken_IsRandom(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		r := GenerateRawToken()
		if seen[r] {
			t.Fatalf("collision after %d iterations: %q", i, r)
		}
		seen[r] = true
	}
}

func TestHashToken_IsDeterministic(t *testing.T) {
	a := HashToken("abc")
	b := HashToken("abc")
	if a != b {
		t.Errorf("HashToken(\"abc\") not deterministic: %q vs %q", a, b)
	}
	if HashToken("abc") == HashToken("abd") {
		t.Error("HashToken collided on different inputs")
	}
}

func TestConstantTimeEqual(t *testing.T) {
	if !ConstantTimeEqual("abc", "abc") {
		t.Error("equal strings should compare equal")
	}
	if ConstantTimeEqual("abc", "abd") {
		t.Error("unequal strings should compare unequal")
	}
	if ConstantTimeEqual("abc", "abcd") {
		t.Error("different-length strings should compare unequal")
	}
}

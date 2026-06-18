package service

import (
	"testing"

	"github.com/mileusna/useragent"
)

func TestUserAgentLibAvailable(t *testing.T) {
	ua := useragent.Parse("Mozilla/5.0 (iPhone; CPU iPhone OS 10_3_2 like Mac OS X) AppleWebKit/603.2.4 (KHTML, like Gecko) Version/10.0 Mobile/14F89 Safari/602.1")
	if ua.Name != "Safari" {
		t.Fatalf("expected browser Safari, got %q", ua.Name)
	}
	if !ua.Mobile {
		t.Fatalf("expected mobile=true for iPhone UA")
	}
}

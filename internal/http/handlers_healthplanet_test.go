package http

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/healthplanet"
	"github.com/capybara-translation/goblog/internal/repo"
	"github.com/capybara-translation/goblog/internal/service"
)

// fakeAuthClient / fakeTokenRepo: minimal in-package fakes for handler tests.
// fakeAuthClient satisfies service.HealthPlanetAuthClient; fakeTokenRepo
// satisfies repo.HealthPlanetTokenRepository.
type fakeAuthClient struct {
	url         string
	exchangeErr error
	gotCode     string
}

func (f *fakeAuthClient) AuthCodeURL() string { return f.url }
func (f *fakeAuthClient) ExchangeCode(code string) (*healthplanet.Token, error) {
	f.gotCode = code
	if f.exchangeErr != nil {
		return nil, f.exchangeErr
	}
	return &healthplanet.Token{AccessToken: "AT/abc", RefreshToken: "RT/def", ExpiresIn: 2592000}, nil
}

type fakeTokenRepo struct {
	saved bool
}

func (f *fakeTokenRepo) Load() (*domain.HealthPlanetToken, error) { return nil, nil }
func (f *fakeTokenRepo) Save(accessToken, refreshToken string, expiresAt time.Time) error {
	f.saved = true
	return nil
}

var (
	_ service.HealthPlanetAuthClient   = (*fakeAuthClient)(nil)
	_ repo.HealthPlanetTokenRepository = (*fakeTokenRepo)(nil)
)

func TestHealthPlanetStatus_Disabled(t *testing.T) {
	h := NewHealthPlanetHandlers(nil)
	req := httptest.NewRequest("GET", "/api/v1/healthplanet/status", nil)
	rr := httptest.NewRecorder()
	h.HandleStatus(rr, req)

	if rr.Code != 200 {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body["enabled"] != false {
		t.Errorf("enabled = %v, want false", body["enabled"])
	}
}

func TestHealthPlanetStatus_EnabledNotAuthorized(t *testing.T) {
	svc := service.NewHealthPlanetAdminService(&fakeAuthClient{}, &fakeTokenRepo{})
	h := NewHealthPlanetHandlers(svc)
	req := httptest.NewRequest("GET", "/api/v1/healthplanet/status", nil)
	rr := httptest.NewRecorder()
	h.HandleStatus(rr, req)

	if rr.Code != 200 {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body["enabled"] != true || body["authorized"] != false {
		t.Errorf("unexpected body: %v", body)
	}
}

func TestHealthPlanetAuthURL(t *testing.T) {
	svc := service.NewHealthPlanetAdminService(&fakeAuthClient{url: "https://hp.example/auth"}, &fakeTokenRepo{})
	h := NewHealthPlanetHandlers(svc)
	req := httptest.NewRequest("GET", "/api/v1/healthplanet/auth-url", nil)
	rr := httptest.NewRecorder()
	h.HandleAuthURL(rr, req)

	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body["url"] != "https://hp.example/auth" {
		t.Errorf("url = %q", body["url"])
	}
}

func TestHealthPlanetExchange_Success(t *testing.T) {
	client := &fakeAuthClient{}
	tokenRepo := &fakeTokenRepo{}
	svc := service.NewHealthPlanetAdminService(client, tokenRepo)
	h := NewHealthPlanetHandlers(svc)
	req := httptest.NewRequest("POST", "/api/v1/healthplanet/exchange", strings.NewReader(`{"code":"the-code"}`))
	rr := httptest.NewRecorder()
	h.HandleExchange(rr, req)

	if rr.Code != 204 {
		t.Fatalf("status = %d, want 204 (body: %s)", rr.Code, rr.Body.String())
	}
	if client.gotCode != "the-code" {
		t.Errorf("code = %q, want the-code", client.gotCode)
	}
	if !tokenRepo.saved {
		t.Error("token was not saved")
	}
}

func TestHealthPlanetExchange_EmptyCode(t *testing.T) {
	svc := service.NewHealthPlanetAdminService(&fakeAuthClient{}, &fakeTokenRepo{})
	h := NewHealthPlanetHandlers(svc)
	req := httptest.NewRequest("POST", "/api/v1/healthplanet/exchange", strings.NewReader(`{"code":""}`))
	rr := httptest.NewRecorder()
	h.HandleExchange(rr, req)

	if rr.Code != 400 {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestHealthPlanetExchange_ExchangeFails(t *testing.T) {
	svc := service.NewHealthPlanetAdminService(&fakeAuthClient{exchangeErr: errors.New("invalid_grant")}, &fakeTokenRepo{})
	h := NewHealthPlanetHandlers(svc)
	req := httptest.NewRequest("POST", "/api/v1/healthplanet/exchange", strings.NewReader(`{"code":"expired"}`))
	rr := httptest.NewRecorder()
	h.HandleExchange(rr, req)

	if rr.Code != 400 {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body["error"] == "" {
		t.Error("error message should be present")
	}
}

func TestHealthPlanetAuthURL_Disabled(t *testing.T) {
	h := NewHealthPlanetHandlers(nil)
	req := httptest.NewRequest("GET", "/api/v1/healthplanet/auth-url", nil)
	rr := httptest.NewRecorder()
	h.HandleAuthURL(rr, req)

	if rr.Code != 503 {
		t.Fatalf("status = %d, want 503", rr.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body["error"] == "" {
		t.Error("error message should be present")
	}
}

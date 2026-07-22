package service

import (
	"errors"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/healthplanet"
)

type mockHealthPlanetAuthClient struct {
	authCodeURL      string
	exchangeCodeFunc func(code string) (*healthplanet.Token, error)
}

func (m *mockHealthPlanetAuthClient) AuthCodeURL() string {
	return m.authCodeURL
}

func (m *mockHealthPlanetAuthClient) ExchangeCode(code string) (*healthplanet.Token, error) {
	if m.exchangeCodeFunc != nil {
		return m.exchangeCodeFunc(code)
	}
	return &healthplanet.Token{AccessToken: "AT/abc", RefreshToken: "RT/def", ExpiresIn: 2592000}, nil
}

func TestHealthPlanetAdmin_AuthCodeURL(t *testing.T) {
	client := &mockHealthPlanetAuthClient{authCodeURL: "https://example.com/oauth/auth?x=1"}
	svc := NewHealthPlanetAdminService(client, &mockHealthPlanetTokenRepo{})
	if got := svc.AuthCodeURL(); got != "https://example.com/oauth/auth?x=1" {
		t.Errorf("AuthCodeURL = %q", got)
	}
}

func TestHealthPlanetAdmin_Exchange_SavesToken(t *testing.T) {
	var gotCode string
	client := &mockHealthPlanetAuthClient{
		exchangeCodeFunc: func(code string) (*healthplanet.Token, error) {
			gotCode = code
			return &healthplanet.Token{AccessToken: "AT/abc", RefreshToken: "RT/def", ExpiresIn: 2592000}, nil
		},
	}
	tokenRepo := &mockHealthPlanetTokenRepo{}
	svc := NewHealthPlanetAdminService(client, tokenRepo)

	if err := svc.Exchange("the-code"); err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if gotCode != "the-code" {
		t.Errorf("code = %q, want the-code", gotCode)
	}
	if len(tokenRepo.saved) != 1 {
		t.Fatalf("token saved %d times, want 1", len(tokenRepo.saved))
	}
	if tokenRepo.saved[0].accessToken != "AT/abc" || tokenRepo.saved[0].refreshToken != "RT/def" {
		t.Errorf("unexpected saved token: %+v", tokenRepo.saved[0])
	}
	wantExpiry := time.Now().Add(2592000 * time.Second)
	if diff := tokenRepo.saved[0].expiresAt.Sub(wantExpiry); diff < -time.Minute || diff > time.Minute {
		t.Errorf("saved expiresAt = %v, want ~%v", tokenRepo.saved[0].expiresAt, wantExpiry)
	}
}

func TestHealthPlanetAdmin_Exchange_SaveFails(t *testing.T) {
	// Client succeeds but repo.Save fails → error must wrap ErrHealthPlanetTokenSaveFailed.
	client := &mockHealthPlanetAuthClient{}
	tokenRepo := &mockHealthPlanetTokenRepo{saveErr: errors.New("io: write error")}
	svc := NewHealthPlanetAdminService(client, tokenRepo)

	err := svc.Exchange("good-code")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrHealthPlanetTokenSaveFailed) {
		t.Errorf("expected errors.Is(err, ErrHealthPlanetTokenSaveFailed) but got: %v", err)
	}
}

func TestHealthPlanetAdmin_Exchange_ClientError(t *testing.T) {
	client := &mockHealthPlanetAuthClient{
		exchangeCodeFunc: func(string) (*healthplanet.Token, error) {
			return nil, errors.New("invalid_grant")
		},
	}
	tokenRepo := &mockHealthPlanetTokenRepo{}
	svc := NewHealthPlanetAdminService(client, tokenRepo)

	if err := svc.Exchange("expired-code"); err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(tokenRepo.saved) != 0 {
		t.Errorf("token should not be saved on failure, saved %d", len(tokenRepo.saved))
	}
}

func TestHealthPlanetAdmin_Status_NotAuthorized(t *testing.T) {
	svc := NewHealthPlanetAdminService(&mockHealthPlanetAuthClient{}, &mockHealthPlanetTokenRepo{token: nil})
	st, err := svc.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.Authorized {
		t.Error("Authorized = true, want false")
	}
	if st.ExpiresAt != nil || st.LastRefreshedAt != nil {
		t.Errorf("timestamps should be nil when not authorized: %+v", st)
	}
}

func TestHealthPlanetAdmin_Status_Authorized(t *testing.T) {
	tok := validHealthPlanetToken()
	tok.UpdatedAt = time.Now().Add(-time.Hour)
	svc := NewHealthPlanetAdminService(&mockHealthPlanetAuthClient{}, &mockHealthPlanetTokenRepo{token: tok})
	st, err := svc.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !st.Authorized {
		t.Fatal("Authorized = false, want true")
	}
	if st.ExpiresAt == nil || !st.ExpiresAt.Equal(tok.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", st.ExpiresAt, tok.ExpiresAt)
	}
	if st.LastRefreshedAt == nil || !st.LastRefreshedAt.Equal(tok.UpdatedAt) {
		t.Errorf("LastRefreshedAt = %v, want %v", st.LastRefreshedAt, tok.UpdatedAt)
	}
}

package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/capybara-translation/goblog/internal/healthplanet"
	"github.com/capybara-translation/goblog/internal/repo"
)

// ErrHealthPlanetTokenSaveFailed is returned by Exchange when the token was
// obtained from Health Planet successfully but could not be persisted. Callers
// can distinguish this internal failure (→ HTTP 500) from a bad authorization
// code (→ HTTP 400) by testing errors.Is(err, ErrHealthPlanetTokenSaveFailed).
var ErrHealthPlanetTokenSaveFailed = errors.New("failed to store healthplanet token")

// HealthPlanetAuthClient is the part of *healthplanet.Client the admin
// (authorization) flow uses, extracted for mocking.
type HealthPlanetAuthClient interface {
	AuthCodeURL() string
	ExchangeCode(code string) (*healthplanet.Token, error)
}

// Compile-time check: the real client satisfies the interface.
var _ HealthPlanetAuthClient = (*healthplanet.Client)(nil)

// HealthPlanetStatus is what the admin panel shows about the integration.
type HealthPlanetStatus struct {
	Authorized bool
	ExpiresAt  *time.Time
	// LastRefreshedAt is the token row's updated_at. The daily sync
	// refreshes (and re-saves) the token on every successful run, so this
	// doubles as "last successful sync" for display purposes.
	LastRefreshedAt *time.Time
}

// HealthPlanetAdminService backs the admin-panel authorization endpoints.
type HealthPlanetAdminService struct {
	client    HealthPlanetAuthClient
	tokenRepo repo.HealthPlanetTokenRepository
}

func NewHealthPlanetAdminService(client HealthPlanetAuthClient, tokenRepo repo.HealthPlanetTokenRepository) *HealthPlanetAdminService {
	return &HealthPlanetAdminService{client: client, tokenRepo: tokenRepo}
}

// AuthCodeURL returns the Health Planet authorization URL the admin panel
// redirects the operator to.
func (s *HealthPlanetAdminService) AuthCodeURL() string {
	return s.client.AuthCodeURL()
}

// Exchange trades an authorization code for a token and stores it.
func (s *HealthPlanetAdminService) Exchange(code string) error {
	tok, err := s.client.ExchangeCode(code)
	if err != nil {
		return fmt.Errorf("token exchange failed: %w", err)
	}
	expiresAt := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	if err := s.tokenRepo.Save(tok.AccessToken, tok.RefreshToken, expiresAt); err != nil {
		return fmt.Errorf("%w: %v", ErrHealthPlanetTokenSaveFailed, err)
	}
	return nil
}

// Status reports whether a token is stored and its lifecycle timestamps.
func (s *HealthPlanetAdminService) Status() (*HealthPlanetStatus, error) {
	tok, err := s.tokenRepo.Load()
	if err != nil {
		return nil, err
	}
	if tok == nil {
		return &HealthPlanetStatus{}, nil
	}
	expiresAt := tok.ExpiresAt
	updatedAt := tok.UpdatedAt
	return &HealthPlanetStatus{
		Authorized:      true,
		ExpiresAt:       &expiresAt,
		LastRefreshedAt: &updatedAt,
	}, nil
}

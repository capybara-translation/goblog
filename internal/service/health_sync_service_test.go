package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/healthplanet"
	"github.com/capybara-translation/goblog/internal/repo"
)

type mockHealthPlanetClient struct {
	refreshFunc      func(refreshToken string) (*healthplanet.Token, error)
	fetchInnerFunc   func(accessToken string, from, to time.Time) ([]healthplanet.Measurement, error)
	fetchSphygmoFunc func(accessToken string, from, to time.Time) ([]healthplanet.Measurement, error)
}

func (m *mockHealthPlanetClient) Refresh(refreshToken string) (*healthplanet.Token, error) {
	if m.refreshFunc != nil {
		return m.refreshFunc(refreshToken)
	}
	return &healthplanet.Token{AccessToken: "AT/new", RefreshToken: "RT/new", ExpiresIn: 2592000}, nil
}

func (m *mockHealthPlanetClient) FetchInnerscan(accessToken string, from, to time.Time) ([]healthplanet.Measurement, error) {
	if m.fetchInnerFunc != nil {
		return m.fetchInnerFunc(accessToken, from, to)
	}
	return nil, nil
}

func (m *mockHealthPlanetClient) FetchSphygmomanometer(accessToken string, from, to time.Time) ([]healthplanet.Measurement, error) {
	if m.fetchSphygmoFunc != nil {
		return m.fetchSphygmoFunc(accessToken, from, to)
	}
	return nil, nil
}

type mockHealthPlanetTokenRepo struct {
	token   *domain.HealthPlanetToken
	loadErr error
	saved   []savedToken
	saveErr error
}

type savedToken struct {
	accessToken  string
	refreshToken string
	expiresAt    time.Time
}

func (m *mockHealthPlanetTokenRepo) Load() (*domain.HealthPlanetToken, error) {
	return m.token, m.loadErr
}

func (m *mockHealthPlanetTokenRepo) Save(accessToken, refreshToken string, expiresAt time.Time) error {
	m.saved = append(m.saved, savedToken{accessToken, refreshToken, expiresAt})
	return m.saveErr
}

type mockHealthRecordRepo struct {
	upserted  []*domain.HealthRecord
	upsertErr error
}

func (m *mockHealthRecordRepo) Upsert(records []*domain.HealthRecord) error {
	m.upserted = append(m.upserted, records...)
	return m.upsertErr
}

func (m *mockHealthRecordRepo) DailyAverages(fromDate, toDate string) ([]repo.DailyAverage, error) {
	panic("not used in sync tests")
}

func (m *mockHealthRecordRepo) DailyAveragesByDates(dates []string) ([]repo.DailyAverage, error) {
	panic("not used in sync tests")
}

func validHealthPlanetToken() *domain.HealthPlanetToken {
	return &domain.HealthPlanetToken{
		ID:           1,
		AccessToken:  "AT/stored",
		RefreshToken: "RT/stored",
		ExpiresAt:    time.Now().Add(29 * 24 * time.Hour),
	}
}

func TestHealthSync_NoToken(t *testing.T) {
	svc := NewHealthSyncService(&mockHealthPlanetClient{}, &mockHealthPlanetTokenRepo{token: nil}, &mockHealthRecordRepo{})
	err := svc.Sync()
	if !errors.Is(err, ErrHealthPlanetNoToken) {
		t.Fatalf("err = %v, want ErrHealthPlanetNoToken", err)
	}
}

func TestHealthSync_HappyPath(t *testing.T) {
	client := &mockHealthPlanetClient{
		fetchInnerFunc: func(accessToken string, from, to time.Time) ([]healthplanet.Measurement, error) {
			if accessToken != "AT/new" {
				t.Errorf("fetch should use refreshed token, got %q", accessToken)
			}
			if window := to.Sub(from); window != 30*24*time.Hour {
				t.Errorf("window = %v, want 720h", window)
			}
			return []healthplanet.Measurement{
				{Date: "202607201624", Keydata: "72.10", Model: "01000145", Tag: healthplanet.TagWeight},
				{Date: "202607201624", Keydata: "20.80", Model: "01000145", Tag: healthplanet.TagBodyFat},
			}, nil
		},
		fetchSphygmoFunc: func(accessToken string, from, to time.Time) ([]healthplanet.Measurement, error) {
			return []healthplanet.Measurement{
				{Date: "202607210925", Keydata: "119", Model: "00000000", Tag: healthplanet.TagSystolic},
				{Date: "202607210925", Keydata: "82", Model: "00000000", Tag: healthplanet.TagDiastolic},
				{Date: "202607210925", Keydata: "63", Model: "00000000", Tag: healthplanet.TagPulse},
			}, nil
		},
	}
	tokenRepo := &mockHealthPlanetTokenRepo{token: validHealthPlanetToken()}
	recordRepo := &mockHealthRecordRepo{}
	svc := NewHealthSyncService(client, tokenRepo, recordRepo)

	if err := svc.Sync(); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if len(tokenRepo.saved) != 1 {
		t.Fatalf("token saved %d times, want 1", len(tokenRepo.saved))
	}
	if tokenRepo.saved[0].accessToken != "AT/new" || tokenRepo.saved[0].refreshToken != "RT/new" {
		t.Errorf("unexpected saved token: %+v", tokenRepo.saved[0])
	}
	wantExpiry := time.Now().Add(2592000 * time.Second)
	if diff := tokenRepo.saved[0].expiresAt.Sub(wantExpiry); diff < -time.Minute || diff > time.Minute {
		t.Errorf("saved expiresAt = %v, want ~%v", tokenRepo.saved[0].expiresAt, wantExpiry)
	}

	if len(recordRepo.upserted) != 5 {
		t.Fatalf("upserted %d records, want 5", len(recordRepo.upserted))
	}
	weight := recordRepo.upserted[0]
	if weight.Metric != domain.MetricWeight || weight.Value != 72.10 {
		t.Errorf("unexpected first record: %+v", weight)
	}
	wantMeasured := time.Date(2026, 7, 20, 16, 24, 0, 0, time.Local)
	if !weight.MeasuredAt.Equal(wantMeasured) {
		t.Errorf("MeasuredAt = %v, want %v", weight.MeasuredAt, wantMeasured)
	}
}

func TestHealthSync_RefreshFails_AccessStillValid_Continues(t *testing.T) {
	client := &mockHealthPlanetClient{
		refreshFunc: func(string) (*healthplanet.Token, error) {
			return nil, errors.New("healthplanet is down")
		},
		fetchInnerFunc: func(accessToken string, from, to time.Time) ([]healthplanet.Measurement, error) {
			if accessToken != "AT/stored" {
				t.Errorf("should fall back to stored token, got %q", accessToken)
			}
			return nil, nil
		},
	}
	tokenRepo := &mockHealthPlanetTokenRepo{token: validHealthPlanetToken()}
	svc := NewHealthSyncService(client, tokenRepo, &mockHealthRecordRepo{})

	if err := svc.Sync(); err != nil {
		t.Fatalf("Sync should succeed with stored token: %v", err)
	}
	if len(tokenRepo.saved) != 0 {
		t.Errorf("token should not be saved on failed refresh, saved %d", len(tokenRepo.saved))
	}
}

func TestHealthSync_RefreshFails_AccessExpired_ReauthRequired(t *testing.T) {
	client := &mockHealthPlanetClient{
		refreshFunc: func(string) (*healthplanet.Token, error) {
			return nil, errors.New("invalid refresh token")
		},
	}
	expired := validHealthPlanetToken()
	expired.ExpiresAt = time.Now().Add(-time.Hour)
	svc := NewHealthSyncService(client, &mockHealthPlanetTokenRepo{token: expired}, &mockHealthRecordRepo{})

	err := svc.Sync()
	if !errors.Is(err, ErrHealthPlanetReauthRequired) {
		t.Fatalf("err = %v, want ErrHealthPlanetReauthRequired", err)
	}
}

func TestHealthSync_TokenExpiringSoon_SyncsButReturnsSentinel(t *testing.T) {
	// Refresh succeeds but the returned expiry stays within the 7-day warning
	// window — the "refresh does not extend expiry" scenario.
	client := &mockHealthPlanetClient{
		refreshFunc: func(string) (*healthplanet.Token, error) {
			return &healthplanet.Token{AccessToken: "AT/new", RefreshToken: "RT/new", ExpiresIn: 3 * 24 * 3600}, nil
		},
		fetchInnerFunc: func(accessToken string, from, to time.Time) ([]healthplanet.Measurement, error) {
			return []healthplanet.Measurement{
				{Date: "202607201624", Keydata: "72.10", Model: "01000145", Tag: healthplanet.TagWeight},
			}, nil
		},
	}
	tokenRepo := &mockHealthPlanetTokenRepo{token: validHealthPlanetToken()}
	recordRepo := &mockHealthRecordRepo{}
	svc := NewHealthSyncService(client, tokenRepo, recordRepo)

	err := svc.Sync()
	if !errors.Is(err, ErrHealthPlanetTokenExpiringSoon) {
		t.Fatalf("err = %v, want ErrHealthPlanetTokenExpiringSoon", err)
	}
	if len(recordRepo.upserted) != 1 {
		t.Errorf("sync should still store records, upserted %d", len(recordRepo.upserted))
	}
	if len(tokenRepo.saved) != 1 {
		t.Errorf("token should be saved on successful refresh, saved %d times", len(tokenRepo.saved))
	}
}

func TestHealthSync_PartialFetchFailure_SavesOtherAndReturnsError(t *testing.T) {
	client := &mockHealthPlanetClient{
		fetchInnerFunc: func(string, time.Time, time.Time) ([]healthplanet.Measurement, error) {
			return nil, errors.New("innerscan 500")
		},
		fetchSphygmoFunc: func(string, time.Time, time.Time) ([]healthplanet.Measurement, error) {
			return []healthplanet.Measurement{
				{Date: "202607210925", Keydata: "119", Model: "00000000", Tag: healthplanet.TagSystolic},
			}, nil
		},
	}
	recordRepo := &mockHealthRecordRepo{}
	svc := NewHealthSyncService(client, &mockHealthPlanetTokenRepo{token: validHealthPlanetToken()}, recordRepo)

	err := svc.Sync()
	if err == nil {
		t.Fatal("Sync should report the fetch failure")
	}
	if len(recordRepo.upserted) != 1 {
		t.Errorf("successful endpoint's records should be saved, upserted %d", len(recordRepo.upserted))
	}
}

func TestHealthSync_SkipsUnparseableMeasurements(t *testing.T) {
	client := &mockHealthPlanetClient{
		fetchInnerFunc: func(string, time.Time, time.Time) ([]healthplanet.Measurement, error) {
			return []healthplanet.Measurement{
				{Date: "not-a-date", Keydata: "72.10", Tag: healthplanet.TagWeight},
				{Date: "202607201624", Keydata: "not-a-number", Tag: healthplanet.TagWeight},
				{Date: "202607201624", Keydata: "1", Tag: "9999"}, // unknown tag
				{Date: "202607201624", Keydata: "72.10", Tag: healthplanet.TagWeight},
			}, nil
		},
	}
	recordRepo := &mockHealthRecordRepo{}
	svc := NewHealthSyncService(client, &mockHealthPlanetTokenRepo{token: validHealthPlanetToken()}, recordRepo)

	if err := svc.Sync(); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if len(recordRepo.upserted) != 1 {
		t.Errorf("upserted %d records, want 1 (bad entries skipped)", len(recordRepo.upserted))
	}
}

func TestHealthSync_SkipsNonFiniteMeasurements(t *testing.T) {
	// strconv.ParseFloat succeeds for "NaN", "Inf", "-Inf" — they return IEEE
	// non-finite values that SQLite stores as NULL (violating NOT NULL). The
	// non-finite guard must catch them and skip, while the one valid entry is
	// upserted.
	client := &mockHealthPlanetClient{
		fetchInnerFunc: func(string, time.Time, time.Time) ([]healthplanet.Measurement, error) {
			return []healthplanet.Measurement{
				{Date: "202607201624", Keydata: "NaN", Tag: healthplanet.TagWeight},
				{Date: "202607201624", Keydata: "Inf", Tag: healthplanet.TagBodyFat},
				{Date: "202607201624", Keydata: "-Inf", Tag: healthplanet.TagBodyFat},
				{Date: "202607201624", Keydata: "72.10", Tag: healthplanet.TagWeight}, // valid
			}, nil
		},
	}
	recordRepo := &mockHealthRecordRepo{}
	svc := NewHealthSyncService(client, &mockHealthPlanetTokenRepo{token: validHealthPlanetToken()}, recordRepo)

	if err := svc.Sync(); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if len(recordRepo.upserted) != 1 {
		t.Errorf("upserted %d records, want 1 (non-finite entries skipped)", len(recordRepo.upserted))
	}
	if recordRepo.upserted[0].Value != 72.10 {
		t.Errorf("upserted value = %v, want 72.10", recordRepo.upserted[0].Value)
	}
}

func TestHealthSync_FetchFailureAndTokenExpiringSoon_BothReported(t *testing.T) {
	// Refresh succeeds but returns a short ExpiresIn (within the 7-day warning
	// window), AND FetchInnerscan fails. Both conditions must appear in the
	// returned error simultaneously.
	fetchErr := errors.New("innerscan 503")
	client := &mockHealthPlanetClient{
		refreshFunc: func(string) (*healthplanet.Token, error) {
			return &healthplanet.Token{AccessToken: "AT/new", RefreshToken: "RT/new", ExpiresIn: 3 * 24 * 3600}, nil
		},
		fetchInnerFunc: func(string, time.Time, time.Time) ([]healthplanet.Measurement, error) {
			return nil, fetchErr
		},
	}
	tokenRepo := &mockHealthPlanetTokenRepo{token: validHealthPlanetToken()}
	svc := NewHealthSyncService(client, tokenRepo, &mockHealthRecordRepo{})

	err := svc.Sync()
	if err == nil {
		t.Fatal("Sync should return a non-nil error")
	}
	if !errors.Is(err, ErrHealthPlanetTokenExpiringSoon) {
		t.Errorf("err should wrap ErrHealthPlanetTokenExpiringSoon, got: %v", err)
	}
	if !strings.Contains(err.Error(), fetchErr.Error()) {
		t.Errorf("err should mention the fetch failure %q, got: %v", fetchErr.Error(), err)
	}
}

func TestHealthSync_UpsertFailure_ReturnsError(t *testing.T) {
	client := &mockHealthPlanetClient{
		fetchInnerFunc: func(string, time.Time, time.Time) ([]healthplanet.Measurement, error) {
			return []healthplanet.Measurement{
				{Date: "202607201624", Keydata: "72.10", Tag: healthplanet.TagWeight},
			}, nil
		},
	}
	recordRepo := &mockHealthRecordRepo{upsertErr: errors.New("db is locked")}
	svc := NewHealthSyncService(client, &mockHealthPlanetTokenRepo{token: validHealthPlanetToken()}, recordRepo)

	if err := svc.Sync(); err == nil {
		t.Fatal("Sync should surface upsert failure")
	}
}

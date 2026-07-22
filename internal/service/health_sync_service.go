package service

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/healthplanet"
	"github.com/capybara-translation/goblog/internal/repo"
)

// Sentinel errors for hpsync's exit-code and messaging decisions.
var (
	// ErrHealthPlanetNoToken: no token stored yet — authorize from the
	// admin panel (or `hpsync auth` as fallback) first.
	ErrHealthPlanetNoToken = errors.New("no healthplanet token stored")
	// ErrHealthPlanetReauthRequired: refresh failed and the stored access
	// token is expired. Manual re-authorization is the only way out.
	ErrHealthPlanetReauthRequired = errors.New("healthplanet re-authorization required")
	// ErrHealthPlanetTokenExpiringSoon: the sync itself succeeded, but the
	// token expires within the warning window and refreshing did not extend
	// it. Surfaced as a non-zero exit so monitoring fires before the token
	// dies (dying means manual re-auth).
	ErrHealthPlanetTokenExpiringSoon = errors.New("healthplanet token expiring soon")
)

const (
	// healthSyncWindow is how far back each sync looks. Wide on purpose:
	// blood pressure is entered manually and may be backfilled days later;
	// upsert idempotency absorbs the overlap between runs.
	healthSyncWindow = 30 * 24 * time.Hour
	// tokenExpiryWarning triggers ErrHealthPlanetTokenExpiringSoon. One
	// week gives several daily runs' worth of alerts before re-auth is
	// actually required.
	tokenExpiryWarning = 7 * 24 * time.Hour
	// measurementDateFormat is Health Planet's measurement timestamp
	// (minute precision, local time).
	measurementDateFormat = "200601021504"
)

// HealthPlanetClient is the part of *healthplanet.Client that Sync uses,
// extracted for mocking.
type HealthPlanetClient interface {
	Refresh(refreshToken string) (*healthplanet.Token, error)
	FetchInnerscan(accessToken string, from, to time.Time) ([]healthplanet.Measurement, error)
	FetchSphygmomanometer(accessToken string, from, to time.Time) ([]healthplanet.Measurement, error)
}

// Compile-time check: the real client satisfies the interface.
var _ HealthPlanetClient = (*healthplanet.Client)(nil)

// HealthSyncService pulls measurements from Health Planet into the local DB.
type HealthSyncService struct {
	client     HealthPlanetClient
	tokenRepo  repo.HealthPlanetTokenRepository
	recordRepo repo.HealthRecordRepository
}

func NewHealthSyncService(client HealthPlanetClient, tokenRepo repo.HealthPlanetTokenRepository, recordRepo repo.HealthRecordRepository) *HealthSyncService {
	return &HealthSyncService{client: client, tokenRepo: tokenRepo, recordRepo: recordRepo}
}

// Sync refreshes the token, fetches the last 30 days of measurements from
// both endpoints and upserts them. Records from a successful endpoint are
// saved even when the other endpoint fails.
func (s *HealthSyncService) Sync() error {
	tok, err := s.tokenRepo.Load()
	if err != nil {
		return err
	}
	if tok == nil {
		return ErrHealthPlanetNoToken
	}

	now := time.Now()
	accessToken := tok.AccessToken
	expiresAt := tok.ExpiresAt

	newTok, err := s.client.Refresh(tok.RefreshToken)
	if err != nil {
		if now.After(tok.ExpiresAt) {
			return fmt.Errorf("%w: refresh failed (%v) and stored access token expired at %s — re-authorize from the admin panel (/admin/healthplanet) or run 'hpsync auth'",
				ErrHealthPlanetReauthRequired, err, tok.ExpiresAt.Format(time.RFC3339))
		}
		// Transient refresh failure with a still-valid access token: sync
		// with what we have and let a later run refresh.
		log.Printf("warning: healthplanet token refresh failed, using stored token (valid until %s): %v",
			tok.ExpiresAt.Format(time.RFC3339), err)
	} else {
		accessToken = newTok.AccessToken
		expiresAt = now.Add(time.Duration(newTok.ExpiresIn) * time.Second)
		if err := s.tokenRepo.Save(newTok.AccessToken, newTok.RefreshToken, expiresAt); err != nil {
			return fmt.Errorf("failed to save refreshed token: %w", err)
		}
	}

	from, to := now.Add(-healthSyncWindow), now
	var records []*domain.HealthRecord
	var fetchErrs []error

	inner, err := s.client.FetchInnerscan(accessToken, from, to)
	if err != nil {
		fetchErrs = append(fetchErrs, err)
	} else {
		records = append(records, mapHealthMeasurements(inner)...)
	}

	sphygmo, err := s.client.FetchSphygmomanometer(accessToken, from, to)
	if err != nil {
		fetchErrs = append(fetchErrs, err)
	} else {
		records = append(records, mapHealthMeasurements(sphygmo)...)
	}

	if err := s.recordRepo.Upsert(records); err != nil {
		return err
	}
	log.Printf("healthplanet sync: upserted %d records (window %s..%s)",
		len(records), from.Format("2006-01-02"), to.Format("2006-01-02"))

	var errs []error
	if len(fetchErrs) > 0 {
		errs = append(errs, fetchErrs...)
	}
	if expiresAt.Before(now.Add(tokenExpiryWarning)) {
		errs = append(errs, fmt.Errorf("%w: expires at %s — if the next runs do not extend it, re-authorize from the admin panel",
			ErrHealthPlanetTokenExpiringSoon, expiresAt.Format(time.RFC3339)))
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// healthMetricByTag maps Health Planet tags to stored metric names.
var healthMetricByTag = map[string]string{
	healthplanet.TagWeight:    domain.MetricWeight,
	healthplanet.TagBodyFat:   domain.MetricBodyFat,
	healthplanet.TagSystolic:  domain.MetricSystolic,
	healthplanet.TagDiastolic: domain.MetricDiastolic,
	healthplanet.TagPulse:     domain.MetricPulse,
}

// mapHealthMeasurements converts API entries to domain records. Entries that
// fail to parse are skipped with a warning: one malformed entry must not
// abort the whole sync.
func mapHealthMeasurements(ms []healthplanet.Measurement) []*domain.HealthRecord {
	records := make([]*domain.HealthRecord, 0, len(ms))
	for _, m := range ms {
		metric, ok := healthMetricByTag[m.Tag]
		if !ok {
			log.Printf("warning: skipping measurement with unknown tag %q (date %s)", m.Tag, m.Date)
			continue
		}
		measuredAt, err := time.ParseInLocation(measurementDateFormat, m.Date, time.Local)
		if err != nil {
			log.Printf("warning: skipping %s measurement with bad date %q: %v", metric, m.Date, err)
			continue
		}
		value, err := strconv.ParseFloat(m.Keydata, 64)
		if err != nil {
			log.Printf("warning: skipping %s measurement with bad value %q (date %s): %v", metric, m.Keydata, m.Date, err)
			continue
		}
		records = append(records, &domain.HealthRecord{
			MeasuredAt: measuredAt,
			Metric:     metric,
			Value:      value,
		})
	}
	return records
}

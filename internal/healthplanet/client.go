// Package healthplanet is a minimal client for the Tanita Health Planet API
// (OAuth 2.0 + measurement endpoints). Verified behavior as of 2026-07:
// tokens live 30 days, refresh works server-to-server, and neither the
// access token nor the refresh token rotates on refresh.
package healthplanet

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://www.healthplanet.jp"

	// SuccessRedirectURI is Tanita's own success page, used by the CLI
	// fallback (`hpsync auth`): the operator copies the code off the
	// address bar instead of being redirected back to the blog. The web
	// flow passes the blog's own /admin/healthplanet/success URL instead.
	SuccessRedirectURI = "https://www.healthplanet.jp/success.html"

	scope = "innerscan,sphygmomanometer"

	// Measurement tags as defined by the Health Planet API.
	TagWeight    = "6021" // kg
	TagBodyFat   = "6022" // %
	TagSystolic  = "622E" // mmHg
	TagDiastolic = "622F" // mmHg
	TagPulse     = "6230" // bpm

	// timeFormat is the from/to query format (yyyyMMddHHmmss).
	timeFormat = "20060102150405"
)

// Token is the response of the /oauth/token endpoint.
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// Measurement is one entry of the status endpoints' data array: a single
// metric value at a single point in time. A blood-pressure reading arrives
// as three Measurements (systolic/diastolic/pulse) sharing the same Date.
type Measurement struct {
	Date    string `json:"date"`    // "202607201624" — minute precision, local time
	Keydata string `json:"keydata"` // numeric string
	Model   string `json:"model"`   // "00000000" for manual entries
	Tag     string `json:"tag"`
}

type statusResponse struct {
	Data []Measurement `json:"data"`
}

// Client calls the Health Planet API. baseURL is injectable for tests;
// redirectURI must match the one used to obtain the authorization code
// (OAuth requires the same value on the code and token requests).
type Client struct {
	baseURL      string
	clientID     string
	clientSecret string
	redirectURI  string
	httpClient   *http.Client
}

func NewClient(baseURL, clientID, clientSecret, redirectURI string) *Client {
	return &Client{
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// AuthCodeURL returns the URL the operator opens in a browser to authorize
// the app.
func (c *Client) AuthCodeURL() string {
	q := url.Values{
		"client_id":     {c.clientID},
		"redirect_uri":  {c.redirectURI},
		"scope":         {scope},
		"response_type": {"code"},
	}
	return c.baseURL + "/oauth/auth?" + q.Encode()
}

// ExchangeCode trades an authorization code for a token. Codes expire 10
// minutes after issuance.
func (c *Client) ExchangeCode(code string) (*Token, error) {
	return c.requestToken(url.Values{
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"redirect_uri":  {c.redirectURI},
		"code":          {code},
		"grant_type":    {"authorization_code"},
	})
}

// Refresh obtains a fresh token without browser interaction.
func (c *Client) Refresh(refreshToken string) (*Token, error) {
	return c.requestToken(url.Values{
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"redirect_uri":  {c.redirectURI},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	})
}

func (c *Client) requestToken(form url.Values) (*Token, error) {
	resp, err := c.httpClient.PostForm(c.baseURL+"/oauth/token", form)
	if err != nil {
		return nil, fmt.Errorf("healthplanet token request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("healthplanet token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("healthplanet token request failed: status %d: %s", resp.StatusCode, body)
	}
	var tok Token
	if err := json.Unmarshal(body, &tok); err != nil {
		return nil, fmt.Errorf("healthplanet token response not JSON: %w: %s", err, body)
	}
	// Health Planet may answer 200 with an error body; treat a missing
	// access_token as failure rather than storing an empty token.
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("healthplanet token response has no access_token: %s", body)
	}
	return &tok, nil
}

// FetchInnerscan returns body-composition measurements (weight, body fat)
// whose measurement date is within [from, to].
func (c *Client) FetchInnerscan(accessToken string, from, to time.Time) ([]Measurement, error) {
	return c.fetchStatus("/status/innerscan.json", accessToken,
		strings.Join([]string{TagWeight, TagBodyFat}, ","), from, to)
}

// FetchSphygmomanometer returns blood-pressure measurements (systolic,
// diastolic, pulse) whose measurement date is within [from, to].
func (c *Client) FetchSphygmomanometer(accessToken string, from, to time.Time) ([]Measurement, error) {
	return c.fetchStatus("/status/sphygmomanometer.json", accessToken,
		strings.Join([]string{TagSystolic, TagDiastolic, TagPulse}, ","), from, to)
}

func (c *Client) fetchStatus(path, accessToken, tags string, from, to time.Time) ([]Measurement, error) {
	q := url.Values{
		"access_token": {accessToken},
		"date":         {"1"}, // 1 = filter by measurement date (not registration date)
		"tag":          {tags},
		"from":         {from.Format(timeFormat)},
		"to":           {to.Format(timeFormat)},
	}
	resp, err := c.httpClient.Get(c.baseURL + path + "?" + q.Encode())
	if err != nil {
		return nil, fmt.Errorf("healthplanet %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("healthplanet %s response: %w", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("healthplanet %s failed: status %d: %s", path, resp.StatusCode, body)
	}
	var sr statusResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		return nil, fmt.Errorf("healthplanet %s response not JSON: %w: %s", path, err, body)
	}
	return sr.Data, nil
}

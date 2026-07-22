package healthplanet

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

const testRedirectURI = "https://blog.example.com/admin/healthplanet/success"

func TestAuthCodeURL(t *testing.T) {
	c := NewClient(DefaultBaseURL, "my-id", "my-secret", testRedirectURI)
	u, err := url.Parse(c.AuthCodeURL())
	if err != nil {
		t.Fatalf("AuthCodeURL not parseable: %v", err)
	}
	q := u.Query()
	if got := q.Get("client_id"); got != "my-id" {
		t.Errorf("client_id = %q, want my-id", got)
	}
	if got := q.Get("redirect_uri"); got != testRedirectURI {
		t.Errorf("redirect_uri = %q", got)
	}
	if got := q.Get("scope"); got != "innerscan,sphygmomanometer" {
		t.Errorf("scope = %q", got)
	}
	if got := q.Get("response_type"); got != "code" {
		t.Errorf("response_type = %q", got)
	}
	if u.Path != "/oauth/auth" {
		t.Errorf("path = %q, want /oauth/auth", u.Path)
	}
}

func TestExchangeCode(t *testing.T) {
	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = r.ParseForm()
		gotForm = r.PostForm
		w.Write([]byte(`{"access_token":"AT/abc","expires_in":2592000,"refresh_token":"RT/def"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "my-id", "my-secret", testRedirectURI)
	tok, err := c.ExchangeCode("the-code")
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}
	if tok.AccessToken != "AT/abc" || tok.RefreshToken != "RT/def" || tok.ExpiresIn != 2592000 {
		t.Errorf("unexpected token: %+v", tok)
	}
	if gotForm.Get("grant_type") != "authorization_code" {
		t.Errorf("grant_type = %q", gotForm.Get("grant_type"))
	}
	if gotForm.Get("code") != "the-code" {
		t.Errorf("code = %q", gotForm.Get("code"))
	}
	if gotForm.Get("client_id") != "my-id" || gotForm.Get("client_secret") != "my-secret" {
		t.Errorf("client credentials not sent: %v", gotForm)
	}
	if gotForm.Get("redirect_uri") != testRedirectURI {
		t.Errorf("redirect_uri = %q", gotForm.Get("redirect_uri"))
	}
}

func TestRefresh(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if got := r.PostForm.Get("grant_type"); got != "refresh_token" {
			t.Errorf("grant_type = %q", got)
		}
		if got := r.PostForm.Get("refresh_token"); got != "RT/def" {
			t.Errorf("refresh_token = %q", got)
		}
		w.Write([]byte(`{"access_token":"AT/abc","expires_in":2592000,"refresh_token":"RT/def"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "my-id", "my-secret", testRedirectURI)
	tok, err := c.Refresh("RT/def")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if tok.AccessToken != "AT/abc" {
		t.Errorf("unexpected token: %+v", tok)
	}
}

func TestExchangeCode_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "my-id", "my-secret", testRedirectURI)
	if _, err := c.ExchangeCode("expired-code"); err == nil {
		t.Fatal("expected error for HTTP 400, got nil")
	}
}

func TestExchangeCode_MissingAccessToken(t *testing.T) {
	// Health Planet が 200 でエラーボディを返すケースへの防御
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"error":"invalid_request"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "my-id", "my-secret", testRedirectURI)
	if _, err := c.ExchangeCode("bad-code"); err == nil {
		t.Fatal("expected error for missing access_token, got nil")
	}
}

func TestFetchInnerscan(t *testing.T) {
	fixture, err := os.ReadFile("testdata/innerscan.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/status/innerscan.json" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotQuery = r.URL.Query()
		w.Write(fixture)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "my-id", "my-secret", testRedirectURI)
	from := time.Date(2026, 6, 22, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, 7, 22, 10, 30, 0, 0, time.Local)
	ms, err := c.FetchInnerscan("AT/abc", from, to)
	if err != nil {
		t.Fatalf("FetchInnerscan: %v", err)
	}
	if len(ms) != 2 {
		t.Fatalf("len = %d, want 2", len(ms))
	}
	if ms[0].Tag != TagWeight || ms[0].Keydata != "72.10" || ms[0].Date != "202607201624" {
		t.Errorf("unexpected measurement: %+v", ms[0])
	}
	if gotQuery.Get("access_token") != "AT/abc" {
		t.Errorf("access_token = %q", gotQuery.Get("access_token"))
	}
	if gotQuery.Get("date") != "1" {
		t.Errorf("date = %q, want 1 (measurement date)", gotQuery.Get("date"))
	}
	if gotQuery.Get("tag") != "6021,6022" {
		t.Errorf("tag = %q", gotQuery.Get("tag"))
	}
	if gotQuery.Get("from") != "20260622000000" {
		t.Errorf("from = %q", gotQuery.Get("from"))
	}
	if gotQuery.Get("to") != "20260722103000" {
		t.Errorf("to = %q", gotQuery.Get("to"))
	}
}

func TestFetchSphygmomanometer(t *testing.T) {
	fixture, err := os.ReadFile("testdata/sphygmomanometer.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/status/sphygmomanometer.json" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("tag"); got != "622E,622F,6230" {
			t.Errorf("tag = %q", got)
		}
		w.Write(fixture)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "my-id", "my-secret", testRedirectURI)
	ms, err := c.FetchSphygmomanometer("AT/abc", time.Now().Add(-30*24*time.Hour), time.Now())
	if err != nil {
		t.Fatalf("FetchSphygmomanometer: %v", err)
	}
	if len(ms) != 6 {
		t.Fatalf("len = %d, want 6", len(ms))
	}
}

func TestFetchInnerscan_TransportErrorRedactsToken(t *testing.T) {
	// Point at a closed server so the HTTP GET fails at the transport layer.
	// The resulting *url.Error would normally embed the full URL (including
	// access_token as a query param); the fix must strip the query string.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := srv.URL
	srv.Close() // close immediately so the connection is refused

	c := NewClient(addr, "my-id", "my-secret", testRedirectURI)
	_, err := c.FetchInnerscan("SECRET-TOKEN", time.Now().Add(-time.Hour), time.Now())
	if err == nil {
		t.Fatal("expected transport error, got nil")
	}
	if strings.Contains(err.Error(), "SECRET-TOKEN") {
		t.Errorf("access_token leaked in error message: %v", err)
	}
}

func TestFetchInnerscan_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "my-id", "my-secret", testRedirectURI)
	_, err := c.FetchInnerscan("expired", time.Now().Add(-time.Hour), time.Now())
	if err == nil {
		t.Fatal("expected error for HTTP 401, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should mention status code: %v", err)
	}
}

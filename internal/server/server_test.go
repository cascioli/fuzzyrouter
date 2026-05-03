package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/simlimone/fuzzyrouter/internal/matcher"
)

func newTestServer(candidates []string, threshold float64) *Server {
	return New(Options{
		Port:         8080,
		BaseDomain:   "example.com",
		RedirectCode: 302,
		Matcher:      matcher.NewLevenshtein(candidates, threshold),
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
}

func TestHandleRequest_Redirect(t *testing.T) {
	s := newTestServer([]string{"app", "admin", "api"}, 0.5)

	tests := []struct {
		host       string
		wantStatus int
		wantLoc    string
	}{
		{"atp.example.com", http.StatusFound, "http://app.example.com/"},
		{"adnin.example.com", http.StatusFound, "http://admin.example.com/"},
		{"apii.example.com", http.StatusFound, "http://api.example.com/"},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://"+tt.host+"/", nil)
			req.Host = tt.host
			rec := httptest.NewRecorder()

			s.handleRequest(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if loc := rec.Header().Get("Location"); loc != tt.wantLoc {
				t.Errorf("Location = %q, want %q", loc, tt.wantLoc)
			}
		})
	}
}

func TestHandleRequest_NoMatch(t *testing.T) {
	// High threshold → nothing matches "xyz"
	s := newTestServer([]string{"app", "admin"}, 0.9)

	req := httptest.NewRequest(http.MethodGet, "http://xyz.example.com/", nil)
	req.Host = "xyz.example.com"
	rec := httptest.NewRecorder()

	s.handleRequest(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleRequest_NoSubdomain(t *testing.T) {
	s := newTestServer([]string{"app"}, 0.5)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Host = "example.com"
	rec := httptest.NewRecorder()

	s.handleRequest(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("bare base domain: status = %d, want 404", rec.Code)
	}
}

func TestHandleRequest_PreservesPathAndQuery(t *testing.T) {
	s := newTestServer([]string{"app"}, 0.5)

	req := httptest.NewRequest(http.MethodGet, "http://atp.example.com/dashboard?tab=1", nil)
	req.Host = "atp.example.com"
	rec := httptest.NewRecorder()

	s.handleRequest(rec, req)

	loc := rec.Header().Get("Location")
	want := "http://app.example.com/dashboard?tab=1"
	if loc != want {
		t.Errorf("Location = %q, want %q", loc, want)
	}
}

func TestHandleRequest_RedirectCode301(t *testing.T) {
	s := New(Options{
		Port:         8080,
		BaseDomain:   "example.com",
		RedirectCode: 301,
		Matcher:      matcher.NewLevenshtein([]string{"app"}, 0.5),
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	req := httptest.NewRequest(http.MethodGet, "http://atp.example.com/", nil)
	req.Host = "atp.example.com"
	rec := httptest.NewRecorder()

	s.handleRequest(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Errorf("status = %d, want 301", rec.Code)
	}
}

func TestHandleHealth(t *testing.T) {
	s := newTestServer([]string{"app"}, 0.5)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	s.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("health status = %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); body != `{"status":"ok"}` {
		t.Errorf("health body = %q", body)
	}
}

func TestExtractSubdomain(t *testing.T) {
	tests := []struct {
		host   string
		base   string
		want   string
	}{
		{"atp.example.com", "example.com", "atp"},
		{"example.com", "example.com", ""},
		{"other.org", "example.com", ""},
		{"a.b.example.com", "example.com", ""},  // nested — rejected
		{"app.example.com", "example.com", "app"},
	}
	for _, tt := range tests {
		got := extractSubdomain(tt.host, tt.base)
		if got != tt.want {
			t.Errorf("extractSubdomain(%q, %q) = %q, want %q", tt.host, tt.base, got, tt.want)
		}
	}
}

func TestBuildRedirectURL_XForwardedProto(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://atp.example.com/path", nil)
	req.Header.Set("X-Forwarded-Proto", "https")

	got := buildRedirectURL(req, "app", "example.com")
	want := "https://app.example.com/path"
	if got != want {
		t.Errorf("buildRedirectURL = %q, want %q", got, want)
	}
}

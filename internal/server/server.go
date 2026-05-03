// Package server implements the FuzzyRouter HTTP handler and lifecycle management.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/simlimone/fuzzyrouter/internal/matcher"
)

// Options configures the Server. Constructed by main and injected here
// to keep server independent of the config package.
type Options struct {
	Port         int
	BaseDomain   string
	RedirectCode int
	Matcher      matcher.Matcher
	Logger       *slog.Logger
}

// Server wraps an http.Server with fuzzy-routing logic.
type Server struct {
	opts Options
	http *http.Server
}

// New creates a Server and registers routes.
func New(opts Options) *Server {
	s := &Server{opts: opts}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRequest)
	mux.HandleFunc("/healthz", s.handleHealth)

	s.http = &http.Server{
		Addr:         fmt.Sprintf(":%d", opts.Port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return s
}

// Run starts the HTTP server. Blocks until the server stops.
// Returns http.ErrServerClosed on clean shutdown.
func (s *Server) Run() error {
	s.opts.Logger.Info("server starting", "addr", s.http.Addr, "base_domain", s.opts.BaseDomain)
	return s.http.ListenAndServe()
}

// Shutdown performs a graceful shutdown within the given deadline.
func (s *Server) Shutdown(ctx context.Context) error {
	s.opts.Logger.Info("server shutting down")
	return s.http.Shutdown(ctx)
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	// Strip port if present (e.g. "atp.example.com:8080" → "atp.example.com")
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	subdomain := extractSubdomain(host, s.opts.BaseDomain)
	if subdomain == "" {
		s.opts.Logger.Debug("no subdomain", "host", host)
		http.NotFound(w, r)
		return
	}

	match, score := s.opts.Matcher.Match(subdomain)
	if match == "" {
		s.opts.Logger.Warn("no match found",
			"subdomain", subdomain,
			"remote_addr", r.RemoteAddr,
		)
		http.NotFound(w, r)
		return
	}

	target := buildRedirectURL(r, match, s.opts.BaseDomain)

	s.opts.Logger.Info("redirect",
		"from", subdomain,
		"to", match,
		"score", fmt.Sprintf("%.3f", score),
		"target", target,
		"method", r.Method,
		"remote_addr", r.RemoteAddr,
		"user_agent", r.UserAgent(),
	)

	http.Redirect(w, r, target, s.opts.RedirectCode)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"ok"}`)
}

// extractSubdomain strips the base domain suffix and returns the subdomain part.
// Returns empty string if host is the bare base domain or unrelated.
func extractSubdomain(host, baseDomain string) string {
	suffix := "." + baseDomain
	if strings.HasSuffix(host, suffix) {
		sub := strings.TrimSuffix(host, suffix)
		// Reject nested subdomains like "a.b.example.com" — only one label expected.
		if sub != "" && !strings.Contains(sub, ".") {
			return sub
		}
	}
	return ""
}

// buildRedirectURL reconstructs the target URL with the matched subdomain.
// Preserves scheme (via X-Forwarded-Proto), path, and query string.
func buildRedirectURL(r *http.Request, subdomain, baseDomain string) string {
	scheme := "https"
	if r.TLS == nil {
		if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else {
			scheme = "http"
		}
	}
	return scheme + "://" + subdomain + "." + baseDomain + r.URL.RequestURI()
}

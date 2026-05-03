package config

import (
	"os"
	"testing"
)

const sampleYAML = `
port: 9090
log_level: debug
redirect_code: 301
base_domain: example.com
match_threshold: 0.6
subdomains:
  - app
  - admin
  - api
`

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "fuzzyrouter-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(f.Name()) })
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestLoadYAML(t *testing.T) {
	path := writeTempYAML(t, sampleYAML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.LogLevel)
	}
	if cfg.RedirectCode != 301 {
		t.Errorf("RedirectCode = %d, want 301", cfg.RedirectCode)
	}
	if cfg.BaseDomain != "example.com" {
		t.Errorf("BaseDomain = %q, want example.com", cfg.BaseDomain)
	}
	if len(cfg.Subdomains) != 3 {
		t.Errorf("Subdomains len = %d, want 3", len(cfg.Subdomains))
	}
	if cfg.MatchThreshold != 0.6 {
		t.Errorf("MatchThreshold = %.2f, want 0.6", cfg.MatchThreshold)
	}
}

func TestEnvOverride(t *testing.T) {
	path := writeTempYAML(t, sampleYAML)

	t.Setenv("FUZZY_PORT", "7777")
	t.Setenv("FUZZY_LOG_LEVEL", "warn")
	t.Setenv("FUZZY_REDIRECT_CODE", "302")
	t.Setenv("FUZZY_BASE_DOMAIN", "override.io")
	t.Setenv("FUZZY_SUBDOMAINS", "x, y, z")
	t.Setenv("FUZZY_THRESHOLD", "0.75")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Port != 7777 {
		t.Errorf("Port = %d, want 7777", cfg.Port)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want warn", cfg.LogLevel)
	}
	if cfg.RedirectCode != 302 {
		t.Errorf("RedirectCode = %d, want 302", cfg.RedirectCode)
	}
	if cfg.BaseDomain != "override.io" {
		t.Errorf("BaseDomain = %q, want override.io", cfg.BaseDomain)
	}
	if len(cfg.Subdomains) != 3 || cfg.Subdomains[0] != "x" {
		t.Errorf("Subdomains = %v, want [x y z]", cfg.Subdomains)
	}
	if cfg.MatchThreshold != 0.75 {
		t.Errorf("MatchThreshold = %.2f, want 0.75", cfg.MatchThreshold)
	}
}

func TestValidationErrors(t *testing.T) {
	base := `base_domain: x.com
subdomains: [a]
`
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "bad port",
			yaml:    base + "port: 99999\nredirect_code: 302\nlog_level: info\nmatch_threshold: 0.5\n",
			wantErr: "port",
		},
		{
			name:    "bad redirect code",
			yaml:    base + "port: 8080\nredirect_code: 200\nlog_level: info\nmatch_threshold: 0.5\n",
			wantErr: "redirect_code",
		},
		{
			name:    "missing base domain",
			yaml:    "subdomains: [a]\nport: 8080\nredirect_code: 302\nlog_level: info\nmatch_threshold: 0.5\n",
			wantErr: "base_domain",
		},
		{
			name:    "bad log level",
			yaml:    base + "port: 8080\nredirect_code: 302\nlog_level: verbose\nmatch_threshold: 0.5\n",
			wantErr: "log_level",
		},
		{
			name:    "threshold out of range",
			yaml:    base + "port: 8080\nredirect_code: 302\nlog_level: info\nmatch_threshold: 1.5\n",
			wantErr: "match_threshold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempYAML(t, tt.yaml)
			_, err := Load(path)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestLoadNoFile(t *testing.T) {
	// No file path: only defaults + env. Must fail validation (no base_domain).
	_, err := Load("")
	if err == nil {
		t.Fatal("expected validation error for empty config, got nil")
	}
}

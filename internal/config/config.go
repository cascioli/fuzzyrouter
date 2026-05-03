// Package config loads FuzzyRouter configuration from a YAML file and environment variables.
// Environment variables take precedence over file values.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the complete runtime configuration for FuzzyRouter.
type Config struct {
	// Server settings
	Port int `yaml:"port"`

	// LogLevel controls verbosity: "debug", "info", "warn", "error"
	LogLevel string `yaml:"log_level"`

	// RedirectCode is the HTTP status used for redirects (301 or 302).
	RedirectCode int `yaml:"redirect_code"`

	// BaseDomain is the root domain appended to matched subdomains, e.g. "example.com".
	BaseDomain string `yaml:"base_domain"`

	// Subdomains is the list of valid subdomains to match against.
	Subdomains []string `yaml:"subdomains"`

	// MatchThreshold is the minimum similarity score [0.0–1.0] to accept a match.
	// Requests below threshold receive a 404 instead of a redirect.
	MatchThreshold float64 `yaml:"match_threshold"`
}

// defaults returns a Config pre-filled with sensible defaults.
func defaults() Config {
	return Config{
		Port:           8080,
		LogLevel:       "info",
		RedirectCode:   302,
		MatchThreshold: 0.5,
	}
}

// Load reads config from path (YAML), then overlays ENV variables.
// If path is empty, only ENV variables and defaults are used.
func Load(path string) (*Config, error) {
	cfg := defaults()

	if path != "" {
		if err := loadYAML(path, &cfg); err != nil {
			return nil, fmt.Errorf("config: load yaml %q: %w", path, err)
		}
	}

	overlayEnv(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config: validation: %w", err)
	}

	return &cfg, nil
}

func loadYAML(path string, cfg *Config) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true) // reject unknown keys
	return dec.Decode(cfg)
}

// overlayEnv applies environment variable overrides onto cfg.
// Recognized variables:
//
//	FUZZY_PORT           - listening port (integer)
//	FUZZY_LOG_LEVEL      - log level string
//	FUZZY_REDIRECT_CODE  - HTTP redirect status code (301 or 302)
//	FUZZY_BASE_DOMAIN    - base domain string
//	FUZZY_SUBDOMAINS     - comma-separated list of valid subdomains
//	FUZZY_THRESHOLD      - match threshold float [0.0–1.0]
func overlayEnv(cfg *Config) {
	if v := os.Getenv("FUZZY_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Port = n
		}
	}
	if v := os.Getenv("FUZZY_LOG_LEVEL"); v != "" {
		cfg.LogLevel = strings.ToLower(v)
	}
	if v := os.Getenv("FUZZY_REDIRECT_CODE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RedirectCode = n
		}
	}
	if v := os.Getenv("FUZZY_BASE_DOMAIN"); v != "" {
		cfg.BaseDomain = v
	}
	if v := os.Getenv("FUZZY_SUBDOMAINS"); v != "" {
		parts := strings.Split(v, ",")
		clean := parts[:0]
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				clean = append(clean, s)
			}
		}
		cfg.Subdomains = clean
	}
	if v := os.Getenv("FUZZY_THRESHOLD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.MatchThreshold = f
		}
	}
}

func validate(cfg *Config) error {
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("port %d out of range [1–65535]", cfg.Port)
	}
	if cfg.RedirectCode != 301 && cfg.RedirectCode != 302 {
		return fmt.Errorf("redirect_code must be 301 or 302, got %d", cfg.RedirectCode)
	}
	if cfg.BaseDomain == "" {
		return fmt.Errorf("base_domain is required")
	}
	if len(cfg.Subdomains) == 0 {
		return fmt.Errorf("subdomains list is empty")
	}
	if cfg.MatchThreshold < 0 || cfg.MatchThreshold > 1 {
		return fmt.Errorf("match_threshold %.2f out of range [0.0–1.0]", cfg.MatchThreshold)
	}
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("log_level %q invalid, must be debug|info|warn|error", cfg.LogLevel)
	}
	return nil
}

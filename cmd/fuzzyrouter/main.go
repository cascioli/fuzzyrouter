package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/simlimone/fuzzyrouter/internal/config"
	"github.com/simlimone/fuzzyrouter/internal/matcher"
	"github.com/simlimone/fuzzyrouter/internal/server"
)

func main() {
	// Docker healthcheck mode: probe /healthz and exit 0/1.
	// Used by docker-compose healthcheck since scratch has no curl/wget.
	if len(os.Args) == 2 && os.Args[1] == "-healthcheck" {
		port := os.Getenv("FUZZY_PORT")
		if port == "" {
			port = "8080"
		}
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/healthz", port))
		if err != nil || resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Config path: FUZZY_CONFIG env → default "config.yaml"
	cfgPath := os.Getenv("FUZZY_CONFIG")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	logger := buildLogger(cfg.LogLevel)

	m := matcher.NewLevenshtein(cfg.Subdomains, cfg.MatchThreshold)

	srv := server.New(server.Options{
		Port:         cfg.Port,
		BaseDomain:   cfg.BaseDomain,
		RedirectCode: cfg.RedirectCode,
		Matcher:      m,
		Logger:       logger,
	})

	// Run server in background goroutine.
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.Run(); !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	// Block until SIGINT / SIGTERM or server error.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		logger.Error("server error", "err", err)
		os.Exit(1)
	case sig := <-quit:
		logger.Info("signal received", "signal", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "err", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}

// buildLogger returns a JSON slog.Logger at the requested level.
func buildLogger(level string) *slog.Logger {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}

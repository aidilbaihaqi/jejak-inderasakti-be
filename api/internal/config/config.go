package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL   string
	RedisURL      string
	JWTSecret     string
	Port          string
	PublicBaseURL string
	Env           string
	LogLevel      slog.Level

	JoinLimitPerMinute int  // join attempts per IP per minute; 0 disables (default off in development)
	TrustProxy         bool // client IP comes from X-Forwarded-For; only set behind Caddy

	AllowedOrigins []string // browser origins allowed to call the API cross-origin (CORS)
}

const defaultJoinLimitPerMinute = 10

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		RedisURL:      os.Getenv("REDIS_URL"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		Port:          getenv("PORT", "8080"),
		PublicBaseURL: strings.TrimRight(getenv("PUBLIC_BASE_URL", "http://localhost:5173"), "/"),
		Env:           getenv("ENV", "development"),
		LogLevel:      parseLevel(getenv("LOG_LEVEL", "info")),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	limit, err := joinLimit(cfg.Env)
	if err != nil {
		return Config{}, err
	}
	cfg.JoinLimitPerMinute = limit
	cfg.TrustProxy = os.Getenv("TRUST_PROXY") == "true"
	cfg.AllowedOrigins = allowedOrigins(cfg.PublicBaseURL)
	return cfg, nil
}

// allowedOrigins reads CORS_ALLOWED_ORIGINS (comma-separated); unset falls back to PublicBaseURL,
// which is correct when the SPA and API still share a domain.
func allowedOrigins(publicBaseURL string) []string {
	raw := os.Getenv("CORS_ALLOWED_ORIGINS")
	if raw == "" {
		return []string{publicBaseURL}
	}
	var origins []string
	for _, o := range strings.Split(raw, ",") {
		if o = strings.TrimSpace(o); o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}

// joinLimit reads RATE_LIMIT_JOIN_PER_MIN; unset means 10, or unlimited in development.
func joinLimit(env string) (int, error) {
	raw := os.Getenv("RATE_LIMIT_JOIN_PER_MIN")
	if raw == "" {
		if env == "development" {
			return 0, nil
		}
		return defaultJoinLimitPerMinute, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 0 {
		return 0, fmt.Errorf("RATE_LIMIT_JOIN_PER_MIN must be a non-negative integer, got %q", raw)
	}
	return limit, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseLevel(s string) slog.Level {
	var level slog.Level
	if err := level.UnmarshalText([]byte(strings.ToUpper(s))); err != nil {
		return slog.LevelInfo
	}
	return level
}

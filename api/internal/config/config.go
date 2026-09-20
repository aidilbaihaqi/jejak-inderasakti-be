package config

import (
	"errors"
	"log/slog"
	"os"
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
}

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
	return cfg, nil
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

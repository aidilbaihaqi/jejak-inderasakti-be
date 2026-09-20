package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/mail"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/config"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store"
)

const (
	bcryptCost        = 12
	minPasswordLength = 8
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("create host failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	email := flag.String("email", "", "host email (required)")
	name := flag.String("name", "", "host display name (required)")
	password := flag.String("password", "", "host password, min 8 characters (required)")
	flag.Parse()

	if err := validate(*email, *name, *password); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := store.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := store.Migrate(ctx, pool); err != nil {
		return err
	}
	normalized := strings.ToLower(strings.TrimSpace(*email))
	if err := store.New(pool).CreateHost(ctx, normalized, strings.TrimSpace(*name), string(hash)); err != nil {
		return fmt.Errorf("save host: %w", err)
	}
	slog.Info("host saved", "email", normalized)
	return nil
}

func validate(email, name, password string) error {
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("invalid -email: %w", err)
	}
	if strings.TrimSpace(name) == "" {
		return errors.New("-name is required")
	}
	if len(password) < minPasswordLength {
		return fmt.Errorf("-password must be at least %d characters", minPasswordLength)
	}
	return nil
}

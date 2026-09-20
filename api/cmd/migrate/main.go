package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/config"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("migrate failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: migrate <up|down|reset|status>")
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
	return store.MigrateCommand(ctx, pool, os.Args[1])
}

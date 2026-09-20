package store

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/db"
)

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	return MigrateCommand(ctx, pool, "up")
}

// MigrateCommand runs one goose action: up, down (one step), reset (down to zero) or status.
func MigrateCommand(ctx context.Context, pool *pgxpool.Pool, action string) error {
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer func() {
		if err := sqlDB.Close(); err != nil {
			slog.Warn("close migration connection", "err", err)
		}
	}()

	goose.SetBaseFS(db.Migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	var err error
	switch action {
	case "up":
		err = goose.UpContext(ctx, sqlDB, "migrations")
	case "down":
		err = goose.DownContext(ctx, sqlDB, "migrations")
	case "reset":
		err = goose.DownToContext(ctx, sqlDB, "migrations", 0)
	case "status":
		err = goose.StatusContext(ctx, sqlDB, "migrations")
	default:
		return fmt.Errorf("unknown migrate action %q (want up, down, reset, status)", action)
	}
	if err != nil {
		return fmt.Errorf("migrate %s: %w", action, err)
	}
	return nil
}

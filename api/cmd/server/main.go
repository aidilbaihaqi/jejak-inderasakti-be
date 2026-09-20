package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/auth"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/config"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/game"
	apihttp "github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/http"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/leaderboard"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/ws"
)

const (
	shutdownTimeout   = 10 * time.Second
	readHeaderTimeout = 5 * time.Second
	redisPingTimeout  = 2 * time.Second
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel})))

	tokens, err := auth.NewTokens(cfg.JWTSecret)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := store.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := store.Migrate(ctx, pool); err != nil {
		return err
	}

	db := store.New(pool)
	board, closeBoard := openBoard(ctx, cfg.RedisURL)
	defer closeBoard()
	rooms := game.NewRegistry(ctx, db, board)
	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: apihttp.NewRouter(apihttp.Deps{
			Store: db, Rooms: rooms, Tokens: tokens, PublicBaseURL: cfg.PublicBaseURL,
			WebSocket: ws.NewHandler(rooms, tokens, cfg.Env == "development"),
		}),
		ReadHeaderTimeout: readHeaderTimeout,
	}
	return serve(ctx, srv, cfg)
}

func serve(ctx context.Context, srv *http.Server, cfg config.Config) error {
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	slog.Info("listening", "port", cfg.Port, "env", cfg.Env)

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// openBoard connects the Redis leaderboard. Redis is disposable, so a missing or unreachable
// Redis only means rankings are computed from memory; the returned func closes the client.
func openBoard(ctx context.Context, url string) (game.Board, func()) {
	noop := func() {}
	if url == "" {
		slog.Warn("REDIS_URL not set, leaderboard uses memory only")
		return nil, noop
	}
	board, err := leaderboard.NewRedis(url)
	if err != nil {
		slog.Warn("invalid REDIS_URL, leaderboard uses memory only", "err", err)
		return nil, noop
	}
	pingCtx, cancel := context.WithTimeout(ctx, redisPingTimeout)
	defer cancel()
	if err := board.Ping(pingCtx); err != nil {
		slog.Warn("redis unreachable, rankings fall back to memory until it is back", "err", err)
	}
	return board, func() {
		if err := board.Close(); err != nil {
			slog.Warn("close redis", "err", err)
		}
	}
}

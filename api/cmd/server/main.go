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
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/ratelimit"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/ws"
	"github.com/redis/go-redis/v9"
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
	redisClient := openRedis(ctx, cfg.RedisURL)
	defer closeRedis(redisClient)
	rooms := game.NewRegistry(ctx, db, newBoard(redisClient))
	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: apihttp.NewRouter(apihttp.Deps{
			Store: db, Rooms: rooms, Tokens: tokens, PublicBaseURL: cfg.PublicBaseURL, TrustProxy: cfg.TrustProxy,
			JoinLimiter:    ratelimit.New(redisClient, cfg.JoinLimitPerMinute, time.Minute),
			WebSocket:      ws.NewHandler(rooms, tokens, cfg.Env == "development", cfg.AllowedOrigins),
			AllowedOrigins: cfg.AllowedOrigins,
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

// openRedis connects to Redis. Redis is disposable, so a missing or unreachable server only means
// rankings and rate limits fall back to memory; it returns nil when REDIS_URL is unusable.
func openRedis(ctx context.Context, url string) *redis.Client {
	if url == "" {
		slog.Warn("REDIS_URL not set, leaderboard and rate limit use memory only")
		return nil
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		slog.Warn("invalid REDIS_URL, leaderboard and rate limit use memory only", "err", err)
		return nil
	}
	client := redis.NewClient(opts)
	pingCtx, cancel := context.WithTimeout(ctx, redisPingTimeout)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		slog.Warn("redis unreachable, falling back to memory until it is back", "err", err)
	}
	return client
}

// newBoard returns a nil interface (not a typed nil) when there is no Redis client.
func newBoard(client *redis.Client) game.Board {
	if client == nil {
		return nil
	}
	return leaderboard.NewRedis(client)
}

func closeRedis(client *redis.Client) {
	if client == nil {
		return
	}
	if err := client.Close(); err != nil {
		slog.Warn("close redis", "err", err)
	}
}

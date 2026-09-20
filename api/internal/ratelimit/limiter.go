// Package ratelimit limits how often one key (an IP address) may act within a time window.
// Counters live in Redis (rl:{key}); if Redis is unavailable a per-process counter takes over,
// so the limit degrades gracefully instead of failing open or closed (docs/05 section 4).
package ratelimit

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const pruneThreshold = 1024

type Limiter struct {
	client *redis.Client // nil means memory only
	limit  int           // requests allowed per window; 0 or less disables limiting
	window time.Duration

	now     func() time.Time
	mu      sync.Mutex
	windows map[string]*window
}

type window struct {
	count   int
	resetAt time.Time
}

func New(client *redis.Client, limit int, windowSize time.Duration) *Limiter {
	return &Limiter{client: client, limit: limit, window: windowSize, now: time.Now, windows: make(map[string]*window)}
}

// Allow counts one attempt for key and reports whether it is within the limit.
func (l *Limiter) Allow(ctx context.Context, key string) bool {
	if l.limit <= 0 {
		return true
	}
	if l.client != nil {
		count, err := l.countInRedis(ctx, "rl:"+key)
		if err == nil {
			return count <= l.limit
		}
		slog.Warn("rate limit counter unavailable, using in-memory counter", "err", err)
	}
	return l.countInMemory(key) <= l.limit
}

// countInRedis increments the counter and sets its expiry on first use, atomically (MULTI/EXEC).
func (l *Limiter) countInRedis(ctx context.Context, key string) (int, error) {
	pipe := l.client.TxPipeline()
	count := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, l.window)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return int(count.Val()), nil
}

func (l *Limiter) countInMemory(key string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if len(l.windows) > pruneThreshold {
		l.pruneExpired(now)
	}
	w, ok := l.windows[key]
	if !ok || !now.Before(w.resetAt) {
		w = &window{resetAt: now.Add(l.window)}
		l.windows[key] = w
	}
	w.count++
	return w.count
}

func (l *Limiter) pruneExpired(now time.Time) {
	for key, w := range l.windows {
		if !now.Before(w.resetAt) {
			delete(l.windows, key)
		}
	}
}

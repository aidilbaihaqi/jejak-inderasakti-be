package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newRedisLimiter(t *testing.T, limit int) (*Limiter, *miniredis.Miniredis) {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Logf("close: %v", err)
		}
	})
	return New(client, limit, time.Minute), server
}

func TestRedisLimitAllowsTenThenBlocks(t *testing.T) { // IT-17
	limiter, _ := newRedisLimiter(t, 10)
	ctx := context.Background()
	for i := 1; i <= 10; i++ {
		if !limiter.Allow(ctx, "join:1.2.3.4") {
			t.Fatalf("attempt %d should be allowed", i)
		}
	}
	if limiter.Allow(ctx, "join:1.2.3.4") {
		t.Error("attempt 11 should be blocked")
	}
	if !limiter.Allow(ctx, "join:5.6.7.8") {
		t.Error("another IP has its own counter")
	}
}

func TestRedisWindowResetsAfterOneMinute(t *testing.T) {
	limiter, server := newRedisLimiter(t, 2)
	ctx := context.Background()
	limiter.Allow(ctx, "join:ip")
	limiter.Allow(ctx, "join:ip")
	if limiter.Allow(ctx, "join:ip") {
		t.Fatal("third attempt should be blocked")
	}
	if ttl := server.TTL("rl:join:ip"); ttl != time.Minute {
		t.Errorf("ttl = %v, want 1m", ttl)
	}
	server.FastForward(time.Minute + time.Second)
	if !limiter.Allow(ctx, "join:ip") {
		t.Error("counter should reset after the window")
	}
}

func TestTTLIsNotExtendedByLaterAttempts(t *testing.T) {
	limiter, server := newRedisLimiter(t, 100)
	ctx := context.Background()
	limiter.Allow(ctx, "join:ip")
	server.FastForward(40 * time.Second)
	limiter.Allow(ctx, "join:ip")
	if ttl := server.TTL("rl:join:ip"); ttl != 20*time.Second {
		t.Errorf("ttl = %v, want 20s (fixed window, not sliding)", ttl)
	}
}

func TestFallsBackToMemoryWhenRedisIsDown(t *testing.T) {
	limiter, server := newRedisLimiter(t, 3)
	server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for i := 1; i <= 3; i++ {
		if !limiter.Allow(ctx, "join:ip") {
			t.Fatalf("attempt %d should be allowed by the in-memory counter", i)
		}
	}
	if limiter.Allow(ctx, "join:ip") {
		t.Error("the in-memory counter must still enforce the limit")
	}
}

func TestMemoryOnlyLimiterAndWindowReset(t *testing.T) {
	limiter := New(nil, 2, time.Minute)
	now := time.Date(2026, 9, 22, 9, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }
	ctx := context.Background()
	limiter.Allow(ctx, "k")
	limiter.Allow(ctx, "k")
	if limiter.Allow(ctx, "k") {
		t.Error("third attempt should be blocked")
	}
	now = now.Add(time.Minute)
	if !limiter.Allow(ctx, "k") {
		t.Error("a new window should allow again")
	}
}

func TestZeroLimitDisablesLimiting(t *testing.T) {
	limiter := New(nil, 0, time.Minute)
	for i := 0; i < 100; i++ {
		if !limiter.Allow(context.Background(), "k") {
			t.Fatal("limiting should be disabled")
		}
	}
}

func TestMemoryCountersArePruned(t *testing.T) {
	limiter := New(nil, 5, time.Minute)
	now := time.Date(2026, 9, 22, 9, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }
	for i := 0; i <= pruneThreshold; i++ {
		limiter.Allow(context.Background(), string(rune('a'+i%26))+time.Duration(i).String())
	}
	now = now.Add(2 * time.Minute)
	limiter.Allow(context.Background(), "fresh")
	if n := len(limiter.windows); n != 1 {
		t.Errorf("expired counters should be pruned, %d remain", n)
	}
}

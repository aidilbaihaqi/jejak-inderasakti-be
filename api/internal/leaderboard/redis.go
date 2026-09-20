// Package leaderboard keeps per-room rankings in a Redis sorted set (key room:{id}:lb).
// Redis is disposable: callers rebuild it from room state, so every method may fail safely.
package leaderboard

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/rank"
)

const keyTTL = 2 * time.Hour

type Redis struct {
	client *redis.Client
}

func NewRedis(url string) (*Redis, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}
	return &Redis{client: redis.NewClient(opts)}, nil
}

func (l *Redis) Ping(ctx context.Context) error {
	return l.client.Ping(ctx).Err()
}

func (l *Redis) Close() error {
	return l.client.Close()
}

// Save writes (or overwrites) the given players' standings.
func (l *Redis) Save(ctx context.Context, roomID string, standings []rank.Standing) error {
	if len(standings) == 0 {
		return nil
	}
	members := make([]redis.Z, len(standings))
	for i, s := range standings {
		members[i] = redis.Z{Score: s.Key(), Member: s.PlayerID}
	}
	key := roomKey(roomID)
	pipe := l.client.Pipeline()
	pipe.ZAdd(ctx, key, members...)
	pipe.Expire(ctx, key, keyTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("save leaderboard: %w", err)
	}
	return nil
}

// Order returns player ids best first.
func (l *Redis) Order(ctx context.Context, roomID string) ([]string, error) {
	ids, err := l.client.ZRevRange(ctx, roomKey(roomID), 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("read leaderboard: %w", err)
	}
	return ids, nil
}

func (l *Redis) Forget(ctx context.Context, roomID, playerID string) error {
	if err := l.client.ZRem(ctx, roomKey(roomID), playerID).Err(); err != nil {
		return fmt.Errorf("remove from leaderboard: %w", err)
	}
	return nil
}

func roomKey(roomID string) string {
	return "room:" + roomID + ":lb"
}

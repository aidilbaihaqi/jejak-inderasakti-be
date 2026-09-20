package leaderboard

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/rank"
)

func newBoard(t *testing.T) (*Redis, *miniredis.Miniredis) {
	t.Helper()
	server := miniredis.RunT(t)
	board, err := NewRedis("redis://" + server.Addr() + "/0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := board.Close(); err != nil {
			t.Logf("close: %v", err)
		}
	})
	return board, server
}

func TestSaveAndOrderRanksByScoreThenCorrectThenTime(t *testing.T) {
	board, _ := newBoard(t)
	ctx := context.Background()
	err := board.Save(ctx, "r1", []rank.Standing{
		{PlayerID: "slow", Score: 1000, Correct: 3, TotalMs: 40000},
		{PlayerID: "fast", Score: 1000, Correct: 3, TotalMs: 20000},
		{PlayerID: "leader", Score: 2500, Correct: 4, TotalMs: 60000},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := board.Order(ctx, "r1")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"leader", "fast", "slow"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSaveOverwritesAndForgetRemoves(t *testing.T) {
	board, _ := newBoard(t)
	ctx := context.Background()
	if err := board.Save(ctx, "r1", []rank.Standing{{PlayerID: "a", Score: 100}, {PlayerID: "b", Score: 200}}); err != nil {
		t.Fatal(err)
	}
	if err := board.Save(ctx, "r1", []rank.Standing{{PlayerID: "a", Score: 900}}); err != nil {
		t.Fatal(err)
	}
	if got, _ := board.Order(ctx, "r1"); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("after overwrite got %v", got)
	}
	if err := board.Forget(ctx, "r1", "a"); err != nil {
		t.Fatal(err)
	}
	if got, _ := board.Order(ctx, "r1"); !reflect.DeepEqual(got, []string{"b"}) {
		t.Errorf("after forget got %v", got)
	}
}

func TestRoomsAreIsolatedAndKeyExpires(t *testing.T) {
	board, server := newBoard(t)
	ctx := context.Background()
	if err := board.Save(ctx, "r1", []rank.Standing{{PlayerID: "a", Score: 1}}); err != nil {
		t.Fatal(err)
	}
	if got, _ := board.Order(ctx, "r2"); len(got) != 0 {
		t.Errorf("other room should be empty, got %v", got)
	}
	if ttl := server.TTL("room:r1:lb"); ttl != keyTTL {
		t.Errorf("ttl = %v, want %v", ttl, keyTTL)
	}
	server.FastForward(keyTTL + time.Second)
	if got, _ := board.Order(ctx, "r1"); len(got) != 0 {
		t.Errorf("expired key should be empty, got %v", got)
	}
}

func TestSaveNothingIsANoop(t *testing.T) {
	board, _ := newBoard(t)
	if err := board.Save(context.Background(), "r1", nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestErrorsWhenRedisIsDown(t *testing.T) {
	board, server := newBoard(t)
	server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := board.Save(ctx, "r1", []rank.Standing{{PlayerID: "a"}}); err == nil {
		t.Error("Save should fail when Redis is down")
	}
	if _, err := board.Order(ctx, "r1"); err == nil {
		t.Error("Order should fail when Redis is down")
	}
}

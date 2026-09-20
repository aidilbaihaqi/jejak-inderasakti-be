package game

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store"
)

// Registry maps room ids to running room goroutines. Rooms are loaded from the database on first
// use, which also brings them back after a restart.
type Registry struct {
	ctx        context.Context
	store      Store
	board      Board
	clock      func() time.Time
	lbInterval time.Duration

	mu    sync.Mutex
	rooms map[string]*Room
}

// NewRegistry creates a registry; board may be nil to keep rankings in memory only.
func NewRegistry(ctx context.Context, st Store, board Board) *Registry {
	return &Registry{ctx: ctx, store: st, board: board, clock: time.Now, lbInterval: DefaultLBInterval, rooms: make(map[string]*Room)}
}

func (g *Registry) AddPlayer(ctx context.Context, roomID string, p store.Player) error {
	room, err := g.Get(ctx, roomID)
	if err != nil {
		return err
	}
	room.AddPlayer(p)
	return nil
}

// Get returns the live room, loading it from the database if needed.
// It returns store.ErrNotFound for unknown or ended rooms.
func (g *Registry) Get(ctx context.Context, roomID string) (*Room, error) {
	if room := g.lookup(roomID); room != nil {
		return room, nil
	}
	data, err := g.store.RoomByID(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if data.Status == StatusEnded {
		return nil, store.ErrNotFound
	}
	if data.Status == StatusRunning && time.Since(data.StartedAt) >= MaxRoomDuration {
		if err := g.store.MarkRoomEnded(ctx, roomID); err != nil {
			return nil, fmt.Errorf("end expired room: %w", err)
		}
		return nil, store.ErrNotFound
	}
	players, err := g.store.PlayersOfRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	questions, err := g.store.QuestionsByIDs(ctx, data.QuestionIDs)
	if err != nil {
		return nil, err
	}
	return g.spawn(data, players, questions), nil
}

func (g *Registry) lookup(roomID string) *Room {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.rooms[roomID]
}

func (g *Registry) spawn(data store.Room, players []store.Player, questions []store.Question) *Room {
	g.mu.Lock()
	defer g.mu.Unlock()
	if existing, ok := g.rooms[data.ID]; ok {
		return existing
	}
	room := newRoom(roomDeps{
		ctx: g.ctx, store: g.store, board: g.board, clock: g.clock, lbInterval: g.lbInterval, onStop: g.forget,
	}, data, players, questions)
	g.rooms[data.ID] = room
	return room
}

func (g *Registry) forget(room *Room) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.rooms[room.id] == room {
		delete(g.rooms, room.id)
	}
}

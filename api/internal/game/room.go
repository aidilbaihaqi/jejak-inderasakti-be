package game

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"time"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store"
)

const (
	StatusLobby   = "lobby"
	StatusRunning = "running"
	StatusEnded   = "ended"

	MaxRoomDuration = 12 * time.Minute
	podiumSize      = 3
	eventBuffer     = 64
	storeTimeout    = 5 * time.Second
)

var (
	ErrRoomClosed    = errors.New("room is closed")
	ErrUnknownPlayer = errors.New("player is not in this room")
	ErrNotHost       = errors.New("not the host of this room")
)

// Message is the {t, d} envelope sent to clients; see contracts/ws.md.
type Message struct {
	T string `json:"t"`
	D any    `json:"d,omitempty"`
}

// Sink is one client connection. Send must never block; Close must let queued messages drain.
type Sink interface {
	Send(Message)
	Close()
}

// Store is the persistence a room needs; *store.Store satisfies it.
type Store interface {
	RoomByID(ctx context.Context, roomID string) (store.Room, error)
	PlayersOfRoom(ctx context.Context, roomID string) ([]store.Player, error)
	MarkRoomStarted(ctx context.Context, roomID string) (bool, error)
	MarkRoomEnded(ctx context.Context, roomID string) error
	DeletePlayer(ctx context.Context, playerID string) error
}

type playerView struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Avatar   int    `json:"avatar"`
	School   string `json:"school"`
	Lang     string `json:"lang"`
}

type playerState struct {
	info  store.Player
	sink  Sink
	score int
}

// Room owns all state of one game room. Every field below the channels is touched only by the
// room goroutine (ADR-003); other goroutines interact through run/call.
type Room struct {
	id, pin, hostID string
	store           Store
	rootCtx         context.Context
	events          chan func()
	done            chan struct{}
	onStop          func(*Room)

	status  string
	players map[string]*playerState
	order   []string
	host    Sink
	timer   *time.Timer
}

func newRoom(ctx context.Context, data store.Room, players []store.Player, st Store, onStop func(*Room)) *Room {
	r := &Room{
		id: data.ID, pin: data.PIN, hostID: data.HostID, store: st, rootCtx: ctx, onStop: onStop,
		events: make(chan func(), eventBuffer), done: make(chan struct{}),
		status: data.Status, players: make(map[string]*playerState, len(players)),
	}
	for _, p := range players {
		r.addPlayer(p)
	}
	if data.Status == StatusRunning {
		r.armDeadline(MaxRoomDuration - time.Since(data.StartedAt))
	}
	go r.loop(ctx)
	return r
}

func (r *Room) ID() string     { return r.id }
func (r *Room) PIN() string    { return r.pin }
func (r *Room) HostID() string { return r.hostID }

func (r *Room) AddPlayer(p store.Player) {
	r.run(func() { r.addPlayerAndNotify(p) })
}

func (r *Room) AttachPlayer(playerID string, s Sink) error {
	return r.call(func() error {
		p, ok := r.players[playerID]
		if !ok {
			return ErrUnknownPlayer
		}
		if p.sink != nil {
			p.sink.Close()
		}
		p.sink = s
		s.Send(r.stateMessage(p))
		return nil
	})
}

func (r *Room) AttachHost(hostID string, s Sink) error {
	return r.call(func() error {
		if hostID != r.hostID {
			return ErrNotHost
		}
		if r.host != nil {
			r.host.Close()
		}
		r.host = s
		s.Send(r.stateMessage(nil))
		return nil
	})
}

func (r *Room) Detach(s Sink) {
	r.run(func() {
		if r.host == s {
			r.host = nil
		}
		for _, p := range r.players {
			if p.sink == s {
				p.sink = nil
			}
		}
	})
}

func (r *Room) StartSession() { r.run(r.startSession) }
func (r *Room) EndSession()   { r.run(r.endSession) }
func (r *Room) Kick(playerID string) {
	r.run(func() { r.kick(playerID) })
}

func (r *Room) loop(ctx context.Context) {
	defer close(r.done) // runs last: once done is closed the room is already out of the registry
	defer r.onStop(r)
	defer r.stopDeadline()
	for {
		select {
		case fn := <-r.events:
			fn()
			if r.status == StatusEnded {
				return
			}
		case <-ctx.Done():
			r.closeAll()
			return
		}
	}
}

func (r *Room) run(fn func()) bool {
	select {
	case r.events <- fn:
		return true
	case <-r.done:
		return false
	}
}

func (r *Room) call(fn func() error) error {
	reply := make(chan error, 1)
	if !r.run(func() { reply <- fn() }) {
		return ErrRoomClosed
	}
	select {
	case err := <-reply:
		return err
	case <-r.done:
		select {
		case err := <-reply:
			return err
		default:
			return ErrRoomClosed
		}
	}
}

func (r *Room) addPlayer(p store.Player) {
	if _, exists := r.players[p.ID]; exists {
		return
	}
	r.players[p.ID] = &playerState{info: p}
	r.order = append(r.order, p.ID)
}

func (r *Room) addPlayerAndNotify(p store.Player) {
	r.addPlayer(p)
	if r.host != nil {
		r.host.Send(Message{T: "player.joined", D: viewOf(p)})
	}
}

func (r *Room) startSession() {
	if r.status != StatusLobby {
		return
	}
	ctx, cancel := context.WithTimeout(r.rootCtx, storeTimeout)
	defer cancel()
	started, err := r.store.MarkRoomStarted(ctx, r.id)
	if err != nil {
		slog.Error("mark room started", "room", r.id, "err", err)
		return
	}
	if !started {
		return
	}
	r.status = StatusRunning
	r.armDeadline(MaxRoomDuration)
	r.broadcast(Message{T: "room.started"})
}

func (r *Room) endSession() {
	if r.status == StatusEnded {
		return
	}
	ctx, cancel := context.WithTimeout(r.rootCtx, storeTimeout)
	defer cancel()
	if err := r.store.MarkRoomEnded(ctx, r.id); err != nil {
		slog.Error("mark room ended", "room", r.id, "err", err)
	}
	r.status = StatusEnded
	r.broadcast(Message{T: "room.ended", D: map[string]any{"podium": r.podium(), "school_lb": []any{}}})
	r.closeAll()
}

func (r *Room) kick(playerID string) {
	p, ok := r.players[playerID]
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.rootCtx, storeTimeout)
	defer cancel()
	if err := r.store.DeletePlayer(ctx, playerID); err != nil {
		slog.Error("delete kicked player", "room", r.id, "player", playerID, "err", err)
		return
	}
	delete(r.players, playerID)
	r.order = removeID(r.order, playerID)
	if p.sink != nil {
		p.sink.Send(Message{T: "player.kicked", D: map[string]string{"reason": "kicked by host"}})
		p.sink.Close()
	}
}

func (r *Room) armDeadline(remaining time.Duration) {
	if remaining <= 0 {
		remaining = time.Millisecond
	}
	r.timer = time.AfterFunc(remaining, func() { r.run(r.endSession) })
}

func (r *Room) stopDeadline() {
	if r.timer != nil {
		r.timer.Stop()
	}
}

func (r *Room) broadcast(m Message) {
	if r.host != nil {
		r.host.Send(m)
	}
	for _, p := range r.players {
		if p.sink != nil {
			p.sink.Send(m)
		}
	}
}

func (r *Room) closeAll() {
	if r.host != nil {
		r.host.Close()
		r.host = nil
	}
	for _, p := range r.players {
		if p.sink != nil {
			p.sink.Close()
			p.sink = nil
		}
	}
}

// stateMessage builds room.state; p is nil when the receiver is the host.
func (r *Room) stateMessage(p *playerState) Message {
	list := make([]playerView, 0, len(r.order))
	for _, id := range r.order {
		list = append(list, viewOf(r.players[id].info))
	}
	state := map[string]any{"status": r.status, "current_index": 0, "score": 0, "streak": 0, "players": list}
	if p != nil {
		state["score"] = p.score
	}
	return Message{T: "room.state", D: state}
}

func (r *Room) podium() []map[string]any {
	ranked := make([]*playerState, 0, len(r.players))
	for _, p := range r.players {
		ranked = append(ranked, p)
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].info.Nickname < ranked[j].info.Nickname
	})
	if len(ranked) > podiumSize {
		ranked = ranked[:podiumSize]
	}
	podium := make([]map[string]any, 0, len(ranked))
	for i, p := range ranked {
		podium = append(podium, map[string]any{"rank": i + 1, "nickname": p.info.Nickname, "avatar": p.info.Avatar, "score": p.score})
	}
	return podium
}

func viewOf(p store.Player) playerView {
	return playerView{ID: p.ID, Nickname: p.Nickname, Avatar: p.Avatar, School: p.School, Lang: p.Lang}
}

func removeID(ids []string, id string) []string {
	for i, v := range ids {
		if v == id {
			return append(ids[:i], ids[i+1:]...)
		}
	}
	return ids
}

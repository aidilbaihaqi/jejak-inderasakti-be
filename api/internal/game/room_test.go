package game

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store"
)

const waitFor = 2 * time.Second

type fakeStore struct {
	mu      sync.Mutex
	rooms   map[string]store.Room
	players map[string][]store.Player
	started []string
	ended   []string
	deleted []string
}

func newFakeStore() *fakeStore {
	return &fakeStore{rooms: map[string]store.Room{}, players: map[string][]store.Player{}}
}

func (f *fakeStore) RoomByID(_ context.Context, id string) (store.Room, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	room, ok := f.rooms[id]
	if !ok {
		return store.Room{}, store.ErrNotFound
	}
	return room, nil
}

func (f *fakeStore) PlayersOfRoom(_ context.Context, id string) ([]store.Player, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]store.Player(nil), f.players[id]...), nil
}

func (f *fakeStore) MarkRoomStarted(_ context.Context, id string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, id)
	return true, nil
}

func (f *fakeStore) MarkRoomEnded(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ended = append(f.ended, id)
	return nil
}

func (f *fakeStore) DeletePlayer(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeStore) count(list func(*fakeStore) []string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(list(f))
}

type fakeSink struct {
	msgs   chan Message
	closed chan struct{}
	once   sync.Once
}

func newFakeSink() *fakeSink {
	return &fakeSink{msgs: make(chan Message, 64), closed: make(chan struct{})}
}

func (s *fakeSink) Send(m Message) { s.msgs <- m }
func (s *fakeSink) Close()         { s.once.Do(func() { close(s.closed) }) }

func (s *fakeSink) expect(t *testing.T, msgType string) Message {
	t.Helper()
	select {
	case m := <-s.msgs:
		if m.T != msgType {
			t.Fatalf("got message %q, want %q", m.T, msgType)
		}
		return m
	case <-time.After(waitFor):
		t.Fatalf("timed out waiting for %q", msgType)
		return Message{}
	}
}

func (s *fakeSink) expectClosed(t *testing.T) {
	t.Helper()
	select {
	case <-s.closed:
	case <-time.After(waitFor):
		t.Fatal("sink was not closed")
	}
}

func (s *fakeSink) expectQuiet(t *testing.T) {
	t.Helper()
	select {
	case m := <-s.msgs:
		t.Fatalf("unexpected message %q", m.T)
	case <-time.After(50 * time.Millisecond):
	}
}

func setup(t *testing.T) (*Registry, *fakeStore, *Room) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	st := newFakeStore()
	reg := NewRegistry(ctx, st)
	data := store.Room{ID: "room-1", PIN: "123456", HostID: "host-1", Jenjang: "SD", Status: StatusLobby}
	st.rooms[data.ID] = data
	reg.Open(data)
	room, err := reg.Get(ctx, data.ID)
	if err != nil {
		t.Fatal(err)
	}
	return reg, st, room
}

func player(id string) store.Player {
	return store.Player{ID: id, Nickname: "nick-" + id, Avatar: 3, Lang: "id", School: "SDN 1"}
}

func TestPlayerJoinNotifiesHost(t *testing.T) {
	_, _, room := setup(t)
	host := newFakeSink()
	if err := room.AttachHost("host-1", host); err != nil {
		t.Fatal(err)
	}
	host.expect(t, "room.state")

	room.AddPlayer(player("p1"))
	msg := host.expect(t, "player.joined")
	if view := msg.D.(playerView); view.ID != "p1" || view.Nickname != "nick-p1" {
		t.Errorf("unexpected player.joined payload %+v", view)
	}
}

func TestAttachPlayerSendsStateAndRejectsStrangers(t *testing.T) {
	_, _, room := setup(t)
	room.AddPlayer(player("p1"))

	sink := newFakeSink()
	if err := room.AttachPlayer("p1", sink); err != nil {
		t.Fatal(err)
	}
	state := sink.expect(t, "room.state").D.(map[string]any)
	if state["status"] != StatusLobby || len(state["players"].([]playerView)) != 1 {
		t.Errorf("unexpected room.state %+v", state)
	}
	if err := room.AttachPlayer("ghost", newFakeSink()); err != ErrUnknownPlayer {
		t.Errorf("unknown player: got %v, want ErrUnknownPlayer", err)
	}
	if err := room.AttachHost("someone-else", newFakeSink()); err != ErrNotHost {
		t.Errorf("wrong host: got %v, want ErrNotHost", err)
	}
}

func TestReconnectReplacesPreviousConnection(t *testing.T) {
	_, _, room := setup(t)
	room.AddPlayer(player("p1"))
	first, second := newFakeSink(), newFakeSink()
	if err := room.AttachPlayer("p1", first); err != nil {
		t.Fatal(err)
	}
	if err := room.AttachPlayer("p1", second); err != nil {
		t.Fatal(err)
	}
	first.expectClosed(t)
	second.expect(t, "room.state")
}

func TestStartSessionBroadcastsOnce(t *testing.T) {
	_, st, room := setup(t)
	host, p1 := newFakeSink(), newFakeSink()
	room.AddPlayer(player("p1"))
	_ = room.AttachHost("host-1", host)
	_ = room.AttachPlayer("p1", p1)
	host.expect(t, "room.state")
	p1.expect(t, "room.state")

	room.StartSession()
	host.expect(t, "room.started")
	p1.expect(t, "room.started")

	room.StartSession()
	host.expectQuiet(t)
	if n := st.count(func(f *fakeStore) []string { return f.started }); n != 1 {
		t.Errorf("MarkRoomStarted called %d times, want 1", n)
	}
}

func TestEndSessionSendsPodiumAndClosesEveryone(t *testing.T) {
	reg, st, room := setup(t)
	host, p1, p2 := newFakeSink(), newFakeSink(), newFakeSink()
	room.AddPlayer(player("p1"))
	room.AddPlayer(player("p2"))
	_ = room.AttachHost("host-1", host)
	_ = room.AttachPlayer("p1", p1)
	_ = room.AttachPlayer("p2", p2)

	room.EndSession()
	for _, s := range []*fakeSink{host, p1, p2} {
		s.expect(t, "room.state")
		ended := s.expect(t, "room.ended").D.(map[string]any)
		if podium := ended["podium"].([]map[string]any); len(podium) != 2 || podium[0]["rank"] != 1 {
			t.Errorf("unexpected podium %+v", podium)
		}
		s.expectClosed(t)
	}
	if n := st.count(func(f *fakeStore) []string { return f.ended }); n != 1 {
		t.Errorf("MarkRoomEnded called %d times, want 1", n)
	}
	select {
	case <-room.done:
	case <-time.After(waitFor):
		t.Fatal("room goroutine did not stop")
	}
	st.mu.Lock()
	st.rooms["room-1"] = store.Room{ID: "room-1", Status: StatusEnded}
	st.mu.Unlock()
	if _, err := reg.Get(context.Background(), "room-1"); err != store.ErrNotFound {
		t.Errorf("ended room should be gone, got %v", err)
	}
	if err := room.AttachPlayer("p1", newFakeSink()); err != ErrRoomClosed {
		t.Errorf("attach after end: got %v, want ErrRoomClosed", err)
	}
}

func TestKickRemovesPlayerAndNotifiesThem(t *testing.T) {
	_, st, room := setup(t)
	kicked := newFakeSink()
	room.AddPlayer(player("p1"))
	_ = room.AttachPlayer("p1", kicked)
	kicked.expect(t, "room.state")

	room.Kick("p1")
	kicked.expect(t, "player.kicked")
	kicked.expectClosed(t)
	if n := st.count(func(f *fakeStore) []string { return f.deleted }); n != 1 {
		t.Errorf("DeletePlayer called %d times, want 1", n)
	}
	if err := room.AttachPlayer("p1", newFakeSink()); err != ErrUnknownPlayer {
		t.Errorf("kicked player reattach: got %v, want ErrUnknownPlayer", err)
	}
}

func TestRunningRoomAutoEndsAtDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	st := newFakeStore()
	st.rooms["r"] = store.Room{ID: "r", HostID: "h", Status: StatusRunning, StartedAt: time.Now().Add(-MaxRoomDuration + 80*time.Millisecond)}
	reg := NewRegistry(ctx, st)
	room, err := reg.Get(ctx, "r")
	if err != nil {
		t.Fatal(err)
	}
	host := newFakeSink()
	if err := room.AttachHost("h", host); err != nil {
		t.Fatal(err)
	}
	host.expect(t, "room.state")
	host.expect(t, "room.ended")
}

func TestGetEndsExpiredRunningRoom(t *testing.T) {
	ctx := context.Background()
	st := newFakeStore()
	st.rooms["r"] = store.Room{ID: "r", Status: StatusRunning, StartedAt: time.Now().Add(-MaxRoomDuration - time.Minute)}
	reg := NewRegistry(ctx, st)
	if _, err := reg.Get(ctx, "r"); err != store.ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
	if n := st.count(func(f *fakeStore) []string { return f.ended }); n != 1 {
		t.Errorf("expired room should be marked ended, got %d calls", n)
	}
}

func TestGetLoadsRoomAndPlayersAfterRestart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	st := newFakeStore()
	st.rooms["r"] = store.Room{ID: "r", HostID: "h", Status: StatusLobby}
	st.players["r"] = []store.Player{player("p1")}
	reg := NewRegistry(ctx, st)

	room, err := reg.Get(ctx, "r")
	if err != nil {
		t.Fatal(err)
	}
	if err := room.AttachPlayer("p1", newFakeSink()); err != nil {
		t.Errorf("player from database should be attachable: %v", err)
	}
	if again, _ := reg.Get(ctx, "r"); again != room {
		t.Error("second Get should return the same live room")
	}
}

func TestConcurrentJoinsAndConnectionsAreRaceFree(t *testing.T) {
	_, _, room := setup(t)
	var wg sync.WaitGroup
	for i := 0; i < 15; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := fmt.Sprintf("p%d", i)
			room.AddPlayer(player(id))
			sink := newFakeSink()
			if err := room.AttachPlayer(id, sink); err != nil {
				t.Errorf("attach %s: %v", id, err)
				return
			}
			room.Detach(sink)
		}()
	}
	host := newFakeSink()
	if err := room.AttachHost("host-1", host); err != nil {
		t.Fatal(err)
	}
	room.StartSession()
	wg.Wait()
}

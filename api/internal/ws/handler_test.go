package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/auth"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/game"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store"
)

const (
	testSecret = "0123456789abcdef0123456789abcdef"
	readWait   = 2 * time.Second
)

type memStore struct {
	mu      sync.Mutex
	rooms   map[string]store.Room
	players map[string][]store.Player
	ended   int
}

func (m *memStore) RoomByID(_ context.Context, id string) (store.Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	room, ok := m.rooms[id]
	if !ok {
		return store.Room{}, store.ErrNotFound
	}
	return room, nil
}
func (m *memStore) PlayersOfRoom(_ context.Context, id string) ([]store.Player, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.players[id], nil
}
func (m *memStore) MarkRoomStarted(context.Context, string) (bool, error) { return true, nil }
func (m *memStore) MarkRoomEnded(context.Context, string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ended++
	return nil
}
func (m *memStore) DeletePlayer(context.Context, string) error { return nil }

type env struct {
	server *httptest.Server
	tokens *auth.Tokens
	store  *memStore
	rooms  *game.Registry
}

func newEnv(t *testing.T) *env {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	tokens, err := auth.NewTokens(testSecret)
	if err != nil {
		t.Fatal(err)
	}
	st := &memStore{
		rooms:   map[string]store.Room{"room-1": {ID: "room-1", PIN: "123456", HostID: "host-1", Status: game.StatusLobby}},
		players: map[string][]store.Player{"room-1": {{ID: "p1", Nickname: "Budi", Avatar: 2, Lang: "id"}}},
	}
	reg := game.NewRegistry(ctx, st)
	mux := http.NewServeMux()
	mux.Handle("/ws", NewHandler(reg, tokens, true))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &env{server: server, tokens: tokens, store: st, rooms: reg}
}

func (e *env) url(query string) string {
	return "ws" + strings.TrimPrefix(e.server.URL, "http") + "/ws?" + query
}

func (e *env) dial(t *testing.T, query string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(e.url(query), nil)
	if err != nil {
		t.Fatalf("dial %s: %v", query, err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Logf("close: %v", err)
		}
	})
	return conn
}

func (e *env) playerConn(t *testing.T) *websocket.Conn {
	t.Helper()
	token, err := e.tokens.IssuePlayer("room-1", "p1")
	if err != nil {
		t.Fatal(err)
	}
	return e.dial(t, "token="+token)
}

func (e *env) hostConn(t *testing.T) *websocket.Conn {
	t.Helper()
	token, err := e.tokens.IssueHost("host-1")
	if err != nil {
		t.Fatal(err)
	}
	return e.dial(t, "token="+token+"&room=room-1")
}

func read(t *testing.T, conn *websocket.Conn) game.Message {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(readWait)); err != nil {
		t.Fatal(err)
	}
	var raw struct {
		T string          `json:"t"`
		D json.RawMessage `json:"d"`
	}
	if err := conn.ReadJSON(&raw); err != nil {
		t.Fatalf("read: %v", err)
	}
	var d any
	if len(raw.D) > 0 {
		if err := json.Unmarshal(raw.D, &d); err != nil {
			t.Fatal(err)
		}
	}
	return game.Message{T: raw.T, D: d}
}

func expect(t *testing.T, conn *websocket.Conn, msgType string) game.Message {
	t.Helper()
	msg := read(t, conn)
	if msg.T != msgType {
		t.Fatalf("got %q (%v), want %q", msg.T, msg.D, msgType)
	}
	return msg
}

func send(t *testing.T, conn *websocket.Conn, msgType string, data any) {
	t.Helper()
	if err := conn.WriteJSON(map[string]any{"t": msgType, "d": data}); err != nil {
		t.Fatal(err)
	}
}

func TestPlayerConnectReceivesRoomState(t *testing.T) {
	conn := newEnv(t).playerConn(t)
	state := expect(t, conn, "room.state").D.(map[string]any)
	players := state["players"].([]any)
	if state["status"] != "lobby" || len(players) != 1 || players[0].(map[string]any)["nickname"] != "Budi" {
		t.Errorf("unexpected room.state %v", state)
	}
}

func TestPingGetsPong(t *testing.T) {
	conn := newEnv(t).playerConn(t)
	expect(t, conn, "room.state")
	send(t, conn, "ping", nil)
	expect(t, conn, "pong")
}

func TestHostStartReachesHostAndPlayers(t *testing.T) {
	e := newEnv(t)
	player, host := e.playerConn(t), e.hostConn(t)
	expect(t, player, "room.state")
	expect(t, host, "room.state")

	send(t, host, "host.start", nil)
	expect(t, host, "room.started")
	expect(t, player, "room.started")
}

func TestPlayersCannotUseHostCommands(t *testing.T) {
	e := newEnv(t)
	player, host := e.playerConn(t), e.hostConn(t)
	expect(t, player, "room.state")
	expect(t, host, "room.state")

	send(t, player, "host.start", nil)
	send(t, player, "ping", nil)
	expect(t, player, "pong") // processed in order, so host.start was already ignored
	send(t, host, "ping", nil)
	expect(t, host, "pong")
}

func TestHostEndSendsPodiumAndClosesConnections(t *testing.T) {
	e := newEnv(t)
	player, host := e.playerConn(t), e.hostConn(t)
	expect(t, player, "room.state")
	expect(t, host, "room.state")

	send(t, host, "host.end", nil)
	ended := expect(t, player, "room.ended").D.(map[string]any)
	if podium := ended["podium"].([]any); len(podium) != 1 {
		t.Errorf("unexpected podium %v", podium)
	}
	expect(t, host, "room.ended")
	if _, _, err := player.ReadMessage(); !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		t.Errorf("player connection should close normally, got %v", err)
	}
}

func TestHostKickDisconnectsPlayer(t *testing.T) {
	e := newEnv(t)
	player, host := e.playerConn(t), e.hostConn(t)
	expect(t, player, "room.state")
	expect(t, host, "room.state")

	send(t, host, "host.kick", map[string]string{"player_id": "p1"})
	expect(t, player, "player.kicked")
	if _, _, err := player.ReadMessage(); err == nil {
		t.Error("kicked player should be disconnected")
	}
}

func TestPlayerJoinedIsPushedToHost(t *testing.T) {
	e := newEnv(t)
	host := e.hostConn(t)
	expect(t, host, "room.state")

	newcomer := store.Player{ID: "p2", Nickname: "Sari", Avatar: 5, Lang: "en"}
	if err := e.rooms.AddPlayer(context.Background(), "room-1", newcomer); err != nil {
		t.Fatal(err)
	}
	joined := expect(t, host, "player.joined").D.(map[string]any)
	if joined["nickname"] != "Sari" || joined["lang"] != "en" {
		t.Errorf("unexpected player.joined %v", joined)
	}
}

func TestRejectsBadConnections(t *testing.T) {
	e := newEnv(t)
	strangerToken, _ := e.tokens.IssuePlayer("room-1", "ghost")
	otherHost, _ := e.tokens.IssueHost("host-2")
	noRoom, _ := e.tokens.IssuePlayer("missing-room", "p1")
	validHost, _ := e.tokens.IssueHost("host-1")

	t.Run("invalid token is refused before upgrade", func(t *testing.T) {
		_, resp, err := websocket.DefaultDialer.Dial(e.url("token=garbage"), nil)
		if err == nil || resp == nil || resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("want 401, got %v %v", resp, err)
		}
	})
	t.Run("unknown room is 404", func(t *testing.T) {
		_, resp, err := websocket.DefaultDialer.Dial(e.url("token="+noRoom), nil)
		if err == nil || resp == nil || resp.StatusCode != http.StatusNotFound {
			t.Errorf("want 404, got %v %v", resp, err)
		}
	})
	t.Run("host without room id is 404", func(t *testing.T) {
		_, resp, err := websocket.DefaultDialer.Dial(e.url("token="+validHost), nil)
		if err == nil || resp == nil || resp.StatusCode != http.StatusNotFound {
			t.Errorf("want 404, got %v %v", resp, err)
		}
	})
	for name, query := range map[string]string{
		"player not in room":   "token=" + strangerToken,
		"host of another room": "token=" + otherHost + "&room=room-1",
	} {
		t.Run(name, func(t *testing.T) {
			conn := e.dial(t, query)
			if _, _, err := conn.ReadMessage(); !websocket.IsCloseError(err, closeUnauthorized) {
				t.Errorf("want close code %d, got %v", closeUnauthorized, err)
			}
		})
	}
}

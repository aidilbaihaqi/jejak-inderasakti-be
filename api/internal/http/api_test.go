package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/auth"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store"
)

const testSecret = "0123456789abcdef0123456789abcdef"

type fakeStore struct {
	host        store.Host
	room        store.Room
	players     int
	joinErr     error
	createErr   error
	schools     []store.School
	pool        []store.QuestionRef
	createdWith store.NewRoom
	joinedWith  store.NewPlayer
	results     []store.ResultRow
	schoolRanks []store.SchoolRank
}

func (f *fakeStore) HostByEmail(_ context.Context, email string) (store.Host, error) {
	if email != f.host.Email {
		return store.Host{}, store.ErrNotFound
	}
	return f.host, nil
}
func (f *fakeStore) TouchHostLogin(context.Context, string) error { return nil }
func (f *fakeStore) CreateRoom(_ context.Context, in store.NewRoom) (store.Room, error) {
	f.createdWith = in
	return f.room, f.createErr
}
func (f *fakeStore) RoomByPIN(_ context.Context, pin string) (store.Room, error) {
	if pin != f.room.PIN {
		return store.Room{}, store.ErrNotFound
	}
	return f.room, nil
}
func (f *fakeStore) PlayerCount(context.Context, string) (int, error) { return f.players, nil }
func (f *fakeStore) JoinRoom(_ context.Context, in store.NewPlayer) (store.Player, error) {
	f.joinedWith = in
	if f.joinErr != nil {
		return store.Player{}, f.joinErr
	}
	return store.Player{ID: "player-1", Nickname: in.Nickname}, nil
}
func (f *fakeStore) RoomByID(_ context.Context, id string) (store.Room, error) {
	if id != f.room.ID {
		return store.Room{}, store.ErrNotFound
	}
	return f.room, nil
}
func (f *fakeStore) RoomResults(context.Context, string) ([]store.ResultRow, error) {
	return f.results, nil
}
func (f *fakeStore) SchoolLeaderboard(context.Context) ([]store.SchoolRank, error) {
	return f.schoolRanks, nil
}
func (f *fakeStore) SearchSchools(context.Context, string) ([]store.School, error) {
	return f.schools, nil
}
func (f *fakeStore) QuestionPool(context.Context) ([]store.QuestionRef, error) { return f.pool, nil }

type fakeRooms struct {
	added []store.Player
}

func (f *fakeRooms) AddPlayer(_ context.Context, _ string, p store.Player) error {
	f.added = append(f.added, p)
	return nil
}

type harness struct {
	server *httptest.Server
	store  *fakeStore
	rooms  *fakeRooms
	tokens *auth.Tokens
}

func fullPool() []store.QuestionRef {
	var pool []store.QuestionRef
	for site := 1; site <= 5; site++ {
		for level := 1; level <= 3; level++ {
			for i := 0; i < 5; i++ {
				pool = append(pool, store.QuestionRef{ID: fmt.Sprintf("Q%d-%d-%d", site, level, i), Site: site, Level: level})
			}
		}
	}
	return pool
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("secret-pass"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	tokens, err := auth.NewTokens(testSecret)
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{
		store: &fakeStore{
			host: store.Host{ID: "host-1", Email: "host@example.com", PasswordHash: string(hash)},
			room: store.Room{ID: "room-1", PIN: "123456", HostID: "host-1", Jenjang: "SD", Status: "lobby"},
			pool: fullPool(),
		},
		rooms: &fakeRooms{}, tokens: tokens,
	}
	h.server = httptest.NewServer(NewRouter(Deps{Store: h.store, Rooms: h.rooms, Tokens: tokens, PublicBaseURL: "https://jejak.test"}))
	t.Cleanup(h.server.Close)
	return h
}

func (h *harness) do(t *testing.T, method, path, bearer string, body any) (int, map[string]any) {
	t.Helper()
	var reader bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = *bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, h.server.URL+path, &reader)
	if err != nil {
		t.Fatal(err)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		decoded = nil
	}
	return resp.StatusCode, decoded
}

func (h *harness) hostToken(t *testing.T) string {
	t.Helper()
	token, err := h.tokens.IssueHost("host-1")
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func validJoin() map[string]any {
	return map[string]any{"nickname": "Budi", "jenjang": "SD", "avatar": 2, "lang": "id"}
}

func TestHealthz(t *testing.T) {
	status, body := newHarness(t).do(t, "GET", "/api/healthz", "", nil)
	if status != 200 || body["status"] != "ok" {
		t.Errorf("got %d %v", status, body)
	}
}

func TestLogin(t *testing.T) {
	h := newHarness(t)
	tests := []struct {
		name       string
		body       any
		wantStatus int
		wantCode   string
	}{
		{"valid", map[string]string{"email": " Host@Example.com ", "password": "secret-pass"}, 200, ""},
		{"wrong password", map[string]string{"email": "host@example.com", "password": "nope"}, 401, "UNAUTHORIZED"},
		{"unknown email", map[string]string{"email": "who@example.com", "password": "secret-pass"}, 401, "UNAUTHORIZED"},
		{"malformed body", "not-an-object", 400, "INVALID_REQUEST"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := h.do(t, "POST", "/api/auth/login", "", tt.body)
			if status != tt.wantStatus || body["code"] != nilIfEmpty(tt.wantCode) {
				t.Fatalf("got %d %v", status, body)
			}
			if tt.wantStatus == 200 {
				claims, err := h.tokens.Parse(body["token"].(string))
				if err != nil || claims.Role != auth.RoleHost || claims.Subject != "host-1" {
					t.Errorf("bad token: %+v %v", claims, err)
				}
			}
		})
	}
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func TestCreateRoom(t *testing.T) {
	h := newHarness(t)
	body := map[string]any{"jenjang": "SD", "consent_confirmed": true}

	status, resp := h.do(t, "POST", "/api/rooms", h.hostToken(t), body)
	if status != 201 || resp["pin"] != "123456" || resp["id"] != "room-1" || resp["qr_url"] != "https://jejak.test/join?pin=123456" {
		t.Fatalf("got %d %v", status, resp)
	}
	if len(h.store.createdWith.QuestionIDs) != 15 || h.store.createdWith.HostID != "host-1" {
		t.Errorf("room created with %+v", h.store.createdWith)
	}
}

func TestCreateRoomShortSessionSelectsTen(t *testing.T) {
	h := newHarness(t)
	h.do(t, "POST", "/api/rooms", h.hostToken(t), map[string]any{"jenjang": "SMA", "short_session": true, "consent_confirmed": true})
	if n := len(h.store.createdWith.QuestionIDs); n != 10 {
		t.Errorf("short session selected %d questions, want 10", n)
	}
}

func TestCreateRoomRejections(t *testing.T) {
	h := newHarness(t)
	guest, _ := h.tokens.IssuePlayer("room-1", "player-1")
	tests := []struct {
		name       string
		bearer     string
		body       map[string]any
		wantStatus int
		wantCode   string
	}{
		{"no token", "", map[string]any{"jenjang": "SD", "consent_confirmed": true}, 401, "UNAUTHORIZED"},
		{"player token is not a host", guest, map[string]any{"jenjang": "SD", "consent_confirmed": true}, 401, "UNAUTHORIZED"},
		{"bad jenjang", h.hostToken(t), map[string]any{"jenjang": "SMK", "consent_confirmed": true}, 400, "INVALID_REQUEST"},
		{"consent missing", h.hostToken(t), map[string]any{"jenjang": "SD"}, 400, "INVALID_REQUEST"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := h.do(t, "POST", "/api/rooms", tt.bearer, tt.body)
			if status != tt.wantStatus || body["code"] != tt.wantCode {
				t.Errorf("got %d %v", status, body)
			}
		})
	}
}

func TestCreateRoomMaxRooms(t *testing.T) {
	h := newHarness(t)
	h.store.createErr = store.ErrMaxRooms
	status, body := h.do(t, "POST", "/api/rooms", h.hostToken(t), map[string]any{"jenjang": "SD", "consent_confirmed": true})
	if status != 409 || body["code"] != "MAX_ROOMS" {
		t.Errorf("got %d %v", status, body)
	}
}

func TestGetRoom(t *testing.T) {
	h := newHarness(t)
	h.store.players = 4
	status, body := h.do(t, "GET", "/api/rooms/123456", "", nil)
	if status != 200 || body["status"] != "lobby" || body["slots_left"] != float64(11) || body["jenjang"] != "SD" {
		t.Errorf("got %d %v", status, body)
	}
	for _, pin := range []string{"000000", "abc", "12345"} {
		status, body := h.do(t, "GET", "/api/rooms/"+pin, "", nil)
		if status != 404 || body["code"] != "ROOM_NOT_FOUND" {
			t.Errorf("pin %q: got %d %v", pin, status, body)
		}
	}
}

func TestJoinRoom(t *testing.T) {
	h := newHarness(t)
	status, body := h.do(t, "POST", "/api/rooms/123456/join", "", validJoin())
	if status != 200 {
		t.Fatalf("got %d %v", status, body)
	}
	claims, err := h.tokens.Parse(body["player_token"].(string))
	if err != nil || claims.Role != auth.RolePlayer || claims.RoomID != "room-1" || claims.Subject != "player-1" {
		t.Errorf("bad player token: %+v %v", claims, err)
	}
	if len(h.rooms.added) != 1 || h.rooms.added[0].ID != "player-1" {
		t.Error("player was not registered with the room")
	}
}

func TestJoinRoomErrors(t *testing.T) {
	tests := []struct {
		name       string
		pin        string
		joinErr    error
		mutate     func(map[string]any)
		wantStatus int
		wantCode   string
	}{
		{"room full", "123456", store.ErrRoomFull, nil, 409, "ROOM_FULL"},
		{"nickname taken", "123456", store.ErrNicknameTaken, nil, 409, "NICKNAME_TAKEN"},
		{"unknown school", "123456", store.ErrInvalidSchool, nil, 400, "INVALID_REQUEST"},
		{"unknown room", "999999", nil, nil, 404, "ROOM_NOT_FOUND"},
		{"empty nickname", "123456", nil, func(b map[string]any) { b["nickname"] = "  " }, 400, "INVALID_REQUEST"},
		{"long nickname", "123456", nil, func(b map[string]any) { b["nickname"] = strings.Repeat("a", 21) }, 400, "INVALID_REQUEST"},
		{"control chars", "123456", nil, func(b map[string]any) { b["nickname"] = "bad\nname" }, 400, "INVALID_REQUEST"},
		{"avatar 0", "123456", nil, func(b map[string]any) { b["avatar"] = 0 }, 400, "INVALID_REQUEST"},
		{"avatar 13", "123456", nil, func(b map[string]any) { b["avatar"] = 13 }, 400, "INVALID_REQUEST"},
		{"bad lang", "123456", nil, func(b map[string]any) { b["lang"] = "fr" }, 400, "INVALID_REQUEST"},
		{"bad jenjang", "123456", nil, func(b map[string]any) { b["jenjang"] = "Umum" }, 400, "INVALID_REQUEST"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			h.store.joinErr = tt.joinErr
			body := validJoin()
			if tt.mutate != nil {
				tt.mutate(body)
			}
			status, resp := h.do(t, "POST", "/api/rooms/"+tt.pin+"/join", "", body)
			if status != tt.wantStatus || resp["code"] != tt.wantCode {
				t.Errorf("got %d %v", status, resp)
			}
			if len(h.rooms.added) != 0 {
				t.Error("failed join must not touch the live room")
			}
		})
	}
}

func TestSearchSchools(t *testing.T) {
	h := newHarness(t)
	jenjang := "SD"
	h.store.schools = []store.School{{ID: 1, Name: "SDN 1", Jenjang: &jenjang}, {ID: 2, Name: "Sekolah lain"}}
	req, _ := http.NewRequestWithContext(context.Background(), "GET", h.server.URL+"/api/schools?q=SDN", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var list []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || len(list) != 2 || list[0]["jenjang"] != "SD" || list[1]["jenjang"] != nil {
		t.Errorf("got %d %v", resp.StatusCode, list)
	}
}

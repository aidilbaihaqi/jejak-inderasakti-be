package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store"
)

func (h *harness) get(t *testing.T, path, bearer string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), "GET", h.server.URL+path, nil)
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, string(body)
}

func TestExportResultsCSV(t *testing.T) { // IT-13
	h := newHarness(t)
	h.store.results = []store.ResultRow{
		{Rank: 1, Nickname: "Ani", School: "SDN 1", Jenjang: "SD", Score: 12750, CorrectCount: 15, TotalMs: 4200},
		{Rank: 2, Nickname: "Budi, Jr", School: "", Jenjang: "SD", Score: 300, CorrectCount: 1, TotalMs: 9000},
	}
	resp, body := h.get(t, "/api/rooms/room-1/results.csv", h.hostToken(t))

	if resp.StatusCode != 200 || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/csv") {
		t.Fatalf("got %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	if cd := resp.Header.Get("Content-Disposition"); cd != `attachment; filename="hasil-123456.csv"` {
		t.Errorf("Content-Disposition = %q", cd)
	}
	want := "\xef\xbb\xbfrank,nickname,school,jenjang,score,correct_count,total_ms\r\n" +
		"1,Ani,SDN 1,SD,12750,15,4200\r\n" +
		"2,\"Budi, Jr\",,SD,300,1,9000\r\n"
	if body != want {
		t.Errorf("csv =\n%q\nwant\n%q", body, want)
	}
}

func TestExportResultsNeutralisesSpreadsheetFormulas(t *testing.T) {
	h := newHarness(t)
	h.store.results = []store.ResultRow{{Rank: 1, Nickname: "=HYPERLINK(1)", School: "+cmd", Jenjang: "SD"}}
	_, body := h.get(t, "/api/rooms/room-1/results.csv", h.hostToken(t))
	if !strings.Contains(body, "'=HYPERLINK") || !strings.Contains(body, ",'+cmd,") {
		t.Errorf("formula characters must be defused, got %q", body)
	}
}

func TestExportResultsAccessControl(t *testing.T) { // IT-14
	h := newHarness(t)
	otherHost, err := h.tokens.IssueHost("someone-else")
	if err != nil {
		t.Fatal(err)
	}
	player, err := h.tokens.IssuePlayer("room-1", "player-1")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, path, bearer, code string
		status                   int
	}{
		{"no token", "/api/rooms/room-1/results.csv", "", "UNAUTHORIZED", 401},
		{"player token", "/api/rooms/room-1/results.csv", player, "UNAUTHORIZED", 401},
		{"another host", "/api/rooms/room-1/results.csv", otherHost, "FORBIDDEN", 403},
		{"unknown room", "/api/rooms/nope/results.csv", h.hostToken(t), "ROOM_NOT_FOUND", 404},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body := h.get(t, tt.path, tt.bearer)
			if resp.StatusCode != tt.status || !strings.Contains(body, tt.code) {
				t.Errorf("got %d %s", resp.StatusCode, body)
			}
		})
	}
}

func TestSchoolLeaderboard(t *testing.T) { // IT-15
	h := newHarness(t)
	h.store.schoolRanks = []store.SchoolRank{{School: "SDN 1", AvgScore: 8123.456, PlayerCount: 5}}
	resp, body := h.get(t, "/api/leaderboard/schools", "")
	if resp.StatusCode != 200 || body != `[{"avg_score":8123.46,"player_count":5,"school":"SDN 1"}]`+"\n" {
		t.Errorf("got %d %q", resp.StatusCode, body)
	}
	h.store.schoolRanks = nil
	if _, empty := h.get(t, "/api/leaderboard/schools", ""); empty != "[]\n" {
		t.Errorf("no schools must be an empty array, got %q", empty)
	}
}

type countingLimiter struct {
	mu   sync.Mutex
	keys []string
	max  int
}

func (c *countingLimiter) Allow(_ context.Context, key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys = append(c.keys, key)
	return len(c.keys) <= c.max
}

func TestJoinIsRateLimitedPerIP(t *testing.T) { // IT-17
	h := newHarness(t)
	limiter := &countingLimiter{max: 2}
	h.server.Config.Handler = NewRouter(Deps{Store: h.store, Rooms: h.rooms, Tokens: h.tokens, JoinLimiter: limiter})

	for _, nickname := range []string{"Na", "Nb"} {
		body := map[string]any{"nickname": nickname, "jenjang": "SD", "avatar": 1, "lang": "id"}
		if status, resp := h.do(t, "POST", "/api/rooms/123456/join", "", body); status != 200 {
			t.Fatalf("%s: %d %v", nickname, status, resp)
		}
	}
	status, body := h.do(t, "POST", "/api/rooms/123456/join", "", validJoin())
	if status != 429 || body["code"] != "RATE_LIMITED" {
		t.Errorf("third attempt: got %d %v", status, body)
	}
	if len(h.rooms.added) != 2 {
		t.Error("a limited request must not reach the room")
	}
	if !strings.HasPrefix(limiter.keys[0], "join:127.0.0.1") {
		t.Errorf("limiter key = %q", limiter.keys[0])
	}
}

func TestClientIP(t *testing.T) {
	request := func(remote, forwarded string) *http.Request {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = remote
		if forwarded != "" {
			r.Header.Set("X-Forwarded-For", forwarded)
		}
		return r
	}
	tests := []struct {
		name  string
		req   *http.Request
		trust bool
		want  string
	}{
		{"direct connection", request("203.0.113.9:5555", ""), false, "203.0.113.9"},
		{"proxy header ignored unless trusted", request("172.18.0.5:5555", "198.51.100.7"), false, "172.18.0.5"},
		{"trusted proxy", request("172.18.0.5:5555", "198.51.100.7"), true, "198.51.100.7"},
		{"spoofed prefix is skipped, last entry wins", request("172.18.0.5:5555", "1.1.1.1, 198.51.100.7"), true, "198.51.100.7"},
		{"trusted but no header falls back to peer", request("172.18.0.5:5555", ""), true, "172.18.0.5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clientIP(tt.req, tt.trust); got != tt.want {
				t.Errorf("clientIP = %q, want %q", got, tt.want)
			}
		})
	}
}

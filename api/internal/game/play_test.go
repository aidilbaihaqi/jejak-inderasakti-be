package game

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/rank"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type fakeBoard struct {
	mu      sync.Mutex
	order   []string
	orderOf error
	saved   [][]rank.Standing
	forgot  []string
}

func (b *fakeBoard) Save(_ context.Context, _ string, s []rank.Standing) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.saved = append(b.saved, s)
	return nil
}

func (b *fakeBoard) Order(context.Context, string) ([]string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.order, b.orderOf
}

func (b *fakeBoard) Forget(_ context.Context, _, playerID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.forgot = append(b.forgot, playerID)
	return nil
}

func (b *fakeBoard) saveCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.saved)
}

// await reads messages until one of the wanted type arrives, ignoring leaderboard noise.
func (s *fakeSink) await(t *testing.T, msgType string) Message {
	t.Helper()
	deadline := time.After(waitFor)
	for {
		select {
		case m := <-s.msgs:
			if m.T == msgType {
				return m
			}
			if m.T != "lb.update" && m.T != "room.state" && m.T != "player.joined" {
				t.Fatalf("waiting for %q, got %q %v", msgType, m.T, m.D)
			}
		case <-deadline:
			t.Fatalf("timed out waiting for %q", msgType)
		}
	}
}

func (s *fakeSink) awaitError(t *testing.T, code string) {
	t.Helper()
	msg := s.await(t, "error")
	if got := msg.D.(map[string]string)["code"]; got != code {
		t.Fatalf("error code = %s, want %s", got, code)
	}
}

func (s *fakeSink) drain() {
	for {
		select {
		case <-s.msgs:
		default:
			return
		}
	}
}

func testQuestions() []store.Question {
	text := func(id, en string) store.Text { return store.Text{ID: id, EN: en} }
	mc := func(id string, level int) store.Question {
		return store.Question{
			ID: id, Site: 1, Level: level, Type: "mc",
			Prompt: text("Soal "+id, "Question "+id), Explanation: text("Penjelasan "+id, "Explanation "+id),
			Options: []store.Option{
				{ID: "opt-a", Label: text("A-id", "A-en")}, {ID: "opt-b", Label: text("B-id", "B-en"), Correct: true},
				{ID: "opt-c", Label: text("C-id", "C-en")}, {ID: "opt-d", Label: text("D-id", "D-en")},
			},
		}
	}
	trueFalse := store.Question{
		ID: "M1-03", Site: 1, Level: 1, Type: "tf",
		Prompt: text("Benar atau salah?", "True or false?"), Explanation: text("Karena benar", "Because true"),
		Options: []store.Option{
			{ID: "opt-a", Label: text("Benar", "True"), Correct: true}, {ID: "opt-b", Label: text("Salah", "False")},
		},
	}
	return []store.Question{mc("M1-01", 1), mc("M1-02", 2), trueFalse}
}

type playFixture struct {
	t       *testing.T
	reg     *Registry
	st      *fakeStore
	board   *fakeBoard
	clock   *fakeClock
	room    *Room
	host    *fakeSink
	sinks   map[string]*fakeSink
	players []store.Player
}

// newPlay builds a lobby room (SMP: 15s easy / 20s medium timers) with the given players attached.
func newPlay(t *testing.T, players ...store.Player) *playFixture {
	t.Helper()
	return newPlayWithInterval(t, time.Millisecond, players...)
}

func newPlayWithInterval(t *testing.T, lbInterval time.Duration, players ...store.Player) *playFixture {
	t.Helper()
	if len(players) == 0 {
		players = []store.Player{player("p1"), player("p2")}
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	st := newFakeStore()
	st.questions = testQuestions()
	st.rooms["room-1"] = store.Room{ID: "room-1", PIN: "123456", HostID: "host-1", Jenjang: "SMP", Status: StatusLobby, QuestionIDs: []string{"M1-01", "M1-02", "M1-03"}}
	st.players["room-1"] = players
	f := &playFixture{t: t, st: st, board: &fakeBoard{}, clock: &fakeClock{now: time.Date(2026, 9, 21, 9, 0, 0, 0, time.UTC)}, sinks: map[string]*fakeSink{}, players: players}
	f.reg = NewRegistry(ctx, st, f.board)
	f.reg.clock = f.clock.Now
	f.reg.lbInterval = lbInterval
	room, err := f.reg.Get(ctx, "room-1")
	if err != nil {
		t.Fatal(err)
	}
	f.room = room
	f.host = newFakeSink()
	if err := room.AttachHost("host-1", f.host); err != nil {
		t.Fatal(err)
	}
	for _, p := range players {
		f.connect(p.ID)
	}
	return f
}

func (f *playFixture) connect(id string) *fakeSink {
	f.t.Helper()
	sink := newFakeSink()
	if err := f.room.AttachPlayer(id, sink); err != nil {
		f.t.Fatal(err)
	}
	f.sinks[id] = sink
	return sink
}

func (f *playFixture) start() {
	f.t.Helper()
	f.room.StartSession()
	f.host.await(f.t, "room.started")
	for _, s := range f.sinks {
		s.await(f.t, "room.started")
	}
}

// show serves the next question to a player and returns the q.show payload.
func (f *playFixture) show(id string) map[string]any {
	f.t.Helper()
	f.room.NextQuestion(id)
	return f.sinks[id].await(f.t, "q.show").D.(map[string]any)
}

func (f *playFixture) answer(id, questionID, option string) map[string]any {
	f.t.Helper()
	f.room.Answer(id, questionID, &option)
	return f.sinks[id].await(f.t, "q.result").D.(map[string]any)
}

func (f *playFixture) results() []store.Result {
	f.st.mu.Lock()
	defer f.st.mu.Unlock()
	return append([]store.Result(nil), f.st.results...)
}

func TestNextQuestionBeforeStartIsRejected(t *testing.T) { // UT-RM-01
	f := newPlay(t)
	f.room.NextQuestion("p1")
	f.sinks["p1"].awaitError(t, codeNotStarted)
	f.room.Answer("p1", "M1-01", strPtr("opt-b"))
	f.sinks["p1"].awaitError(t, codeNotStarted)
}

func strPtr(s string) *string { return &s }

func TestQuestionIsShownWithoutTheAnswerKey(t *testing.T) {
	f := newPlay(t)
	f.start()
	shown := f.show("p1")

	if shown["index"] != 0 || shown["total"] != 3 || shown["limit_ms"] != int64(15000) || shown["prompt"] != "Soal M1-01" {
		t.Errorf("unexpected q.show %v", shown)
	}
	for _, opt := range shown["options"].([]map[string]string) {
		if len(opt) != 2 || opt["id"] == "" || opt["label"] == "" {
			t.Errorf("option must only carry id and label, got %v", opt)
		}
	}
	if _, leaked := shown["correct"]; leaked {
		t.Error("q.show must never contain the answer key")
	}
}

func TestPlayerReceivesTheirOwnLanguage(t *testing.T) {
	english := player("p1")
	english.Lang = "en"
	f := newPlay(t, english)
	f.start()
	shown := f.show("p1")
	if shown["prompt"] != "Question M1-01" {
		t.Errorf("prompt = %v", shown["prompt"])
	}
	result := f.answer("p1", "M1-01", "opt-b")
	if result["explanation"] != "Explanation M1-01" {
		t.Errorf("explanation = %v", result["explanation"])
	}
}

func TestCorrectAnswerScoresAndPersists(t *testing.T) {
	f := newPlay(t)
	f.start()
	f.show("p1")
	f.clock.Advance(time.Second) // inside the 2s reading time
	result := f.answer("p1", "M1-01", "opt-b")

	want := map[string]any{"correct": true, "correct_option_id": "opt-b", "points": 500, "score": 500, "streak": 1, "finished": false}
	for key, value := range want {
		if result[key] != value {
			t.Errorf("q.result[%s] = %v, want %v", key, result[key], value)
		}
	}
	saved := f.results()
	if len(saved) != 1 {
		t.Fatalf("saved %d results, want 1", len(saved))
	}
	got := saved[0]
	if got.PlayerID != "p1" || got.QuestionID != "M1-01" || got.Points != 500 || got.Score != 500 || got.CorrectCount != 1 ||
		got.CurrentIndex != 1 || got.Streak != 1 || got.TotalMs != 1000 || got.Finished || got.Correct == nil || !*got.Correct || got.AnsweredAt == nil {
		t.Errorf("unexpected saved result %+v", got)
	}
}

func TestStreakBonusAndReset(t *testing.T) {
	f := newPlay(t)
	f.start()
	f.show("p1")
	f.answer("p1", "M1-01", "opt-b") // correct, 500

	f.show("p1")
	second := f.answer("p1", "M1-02", "opt-b") // correct, 750 + 50 streak bonus
	if second["points"] != 800 || second["streak"] != 2 {
		t.Errorf("second answer = %v", second)
	}

	f.show("p1")
	third := f.answer("p1", "M1-03", "opt-b") // wrong on the true/false question
	if third["correct"] != false || third["points"] != 0 || third["streak"] != 0 || third["score"] != 1300 {
		t.Errorf("wrong answer = %v", third)
	}
}

func TestDoubleAnswerIsRejectedAndFirstIsKept(t *testing.T) { // UT-RM-02
	f := newPlay(t)
	f.start()
	f.show("p1")
	f.answer("p1", "M1-01", "opt-b")

	f.room.Answer("p1", "M1-01", strPtr("opt-a"))
	f.sinks["p1"].awaitError(t, codeAnswered)
	if n := len(f.results()); n != 1 {
		t.Errorf("saved %d results, want 1", n)
	}
}

func TestAnswerBadInputIsRejectedWithoutConsumingTheQuestion(t *testing.T) {
	f := newPlay(t)
	f.start()
	f.show("p1")

	f.room.Answer("p1", "M1-02", strPtr("opt-b")) // not the active question
	f.sinks["p1"].awaitError(t, codeNotServed)
	f.room.Answer("p1", "M1-01", strPtr("opt-zzz"))
	f.sinks["p1"].awaitError(t, codeBadOption)
	if result := f.answer("p1", "M1-01", "opt-b"); result["correct"] != true {
		t.Errorf("question should still be answerable, got %v", result)
	}
}

func TestAnswerAfterDeadlineScoresZero(t *testing.T) { // UT-RM-03
	f := newPlay(t)
	f.start()
	f.show("p1")
	f.clock.Advance(15*time.Second + time.Second + time.Millisecond) // limit + network tolerance
	result := f.answer("p1", "M1-01", "opt-b")

	if result["points"] != 0 || result["correct"] != false || result["score"] != 0 {
		t.Errorf("late answer = %v", result)
	}
	saved := f.results()[0]
	if saved.Correct != nil || saved.AnsweredAt != nil || saved.OptionID != nil || saved.TotalMs != 15000 {
		t.Errorf("late answer must be stored as a timeout, got %+v", saved)
	}
}

func TestAnswerInsideNetworkToleranceStillScores(t *testing.T) {
	f := newPlay(t)
	f.start()
	f.show("p1")
	f.clock.Advance(15*time.Second + 500*time.Millisecond)
	if result := f.answer("p1", "M1-01", "opt-b"); result["points"] != 300 {
		t.Errorf("answer 500ms after the timer should still score the 60%% floor, got %v", result)
	}
}

func TestTimedOutQuestionIsRecordedWhenNextIsRequested(t *testing.T) { // UT-RM-04
	f := newPlay(t)
	f.start()
	f.show("p1")

	f.room.NextQuestion("p1")
	f.sinks["p1"].awaitError(t, codeInProgress)

	f.clock.Advance(17 * time.Second)
	second := f.show("p1")
	if second["index"] != 1 || second["prompt"] != "Soal M1-02" {
		t.Errorf("expected the next question, got %v", second)
	}
	saved := f.results()
	if len(saved) != 1 || saved[0].QuestionID != "M1-01" || saved[0].Points != 0 || saved[0].Correct != nil || saved[0].Streak != 0 {
		t.Errorf("timeout not recorded correctly: %+v", saved)
	}
}

func TestClientReportedTimeoutCountsAsZero(t *testing.T) {
	f := newPlay(t)
	f.start()
	f.show("p1")
	f.room.Answer("p1", "M1-01", nil)
	result := f.sinks["p1"].await(t, "q.result").D.(map[string]any)
	if result["points"] != 0 || result["correct"] != false {
		t.Errorf("got %v", result)
	}
}

func TestRoomEndsWhenEveryPlayerFinished(t *testing.T) { // UT-RM-06
	f := newPlay(t)
	f.start()
	for _, id := range []string{"p1", "p2"} {
		for _, q := range []string{"M1-01", "M1-02", "M1-03"} {
			f.show(id)
			last := f.answer(id, q, "opt-b")
			if q == "M1-03" && last["finished"] != true {
				t.Errorf("last result should be marked finished, got %v", last)
			}
		}
		if id == "p1" {
			f.host.drain()
			f.room.NextQuestion("p1")
			f.sinks["p1"].awaitError(t, codeFinished)
		}
	}
	ended := f.host.await(t, "room.ended").D.(map[string]any)
	if podium := ended["podium"].([]map[string]any); len(podium) != 2 {
		t.Errorf("podium = %v", podium)
	}
	if n := f.st.count(func(s *fakeStore) []string { return s.ended }); n != 1 {
		t.Errorf("room marked ended %d times, want 1", n)
	}
	if last := f.results()[len(f.results())-1]; !last.Finished {
		t.Error("final result must mark the player as finished")
	}
}

func TestHostEndBroadcastsFinalPodium(t *testing.T) { // UT-RM-07
	f := newPlay(t)
	f.start()
	f.show("p1")
	f.answer("p1", "M1-01", "opt-b")
	f.room.EndSession()
	ended := f.sinks["p2"].await(t, "room.ended").D.(map[string]any)
	podium := ended["podium"].([]map[string]any)
	if podium[0]["nickname"] != "nick-p1" || podium[0]["score"] != 500 || podium[1]["score"] != 0 {
		t.Errorf("podium = %v", podium)
	}
}

func TestReconnectResumesTheActiveQuestion(t *testing.T) { // UT-RM-09
	f := newPlay(t)
	f.start()
	f.show("p1")
	f.room.Detach(f.sinks["p1"])
	f.clock.Advance(5 * time.Second)

	sink := f.connect("p1")
	state := sink.expect(t, "room.state").D.(map[string]any)
	if state["status"] != StatusRunning || state["current_index"] != 0 {
		t.Errorf("room.state = %v", state)
	}
	resumed := sink.expect(t, "q.show").D.(map[string]any)
	if resumed["index"] != 0 || resumed["limit_ms"] != int64(10000) {
		t.Errorf("resumed question = %v", resumed)
	}
	if result := f.answer("p1", "M1-01", "opt-b"); result["correct"] != true {
		t.Errorf("answer after reconnect = %v", result)
	}
}

func TestReconnectAfterDeadlineRecordsTimeout(t *testing.T) { // UT-RM-10
	f := newPlay(t)
	f.start()
	f.show("p1")
	f.room.Detach(f.sinks["p1"])
	f.clock.Advance(20 * time.Second)

	sink := f.connect("p1")
	state := sink.expect(t, "room.state").D.(map[string]any)
	if state["current_index"] != 1 || state["score"] != 0 {
		t.Errorf("room.state = %v", state)
	}
	sink.expectQuiet(t) // no stale question is re-sent
	if saved := f.results(); len(saved) != 1 || saved[0].Points != 0 || saved[0].Correct != nil {
		t.Errorf("timeout not recorded: %+v", saved)
	}
	if next := f.show("p1"); next["index"] != 1 {
		t.Errorf("next question = %v", next)
	}
}

func TestReconnectAfterAnsweringDoesNotReplayQuestion(t *testing.T) {
	f := newPlay(t)
	f.start()
	f.show("p1")
	f.answer("p1", "M1-01", "opt-b")
	f.room.Detach(f.sinks["p1"])

	sink := f.connect("p1")
	state := sink.expect(t, "room.state").D.(map[string]any)
	if state["current_index"] != 1 || state["score"] != 500 || state["streak"] != 1 {
		t.Errorf("room.state = %v", state)
	}
	sink.expectQuiet(t)
}

func TestProgressIsRestoredWhenRoomIsReloaded(t *testing.T) {
	resumed := player("p1")
	resumed.Score, resumed.CorrectCount, resumed.TotalMs, resumed.CurrentIndex, resumed.Streak = 500, 1, 3000, 1, 1
	f := newPlay(t, resumed)
	f.start()
	if shown := f.show("p1"); shown["index"] != 1 || shown["prompt"] != "Soal M1-02" {
		t.Errorf("should continue at the second question, got %v", shown)
	}
	if result := f.answer("p1", "M1-02", "opt-b"); result["score"] != 1300 || result["streak"] != 2 {
		t.Errorf("score must continue from the restored total, got %v", result)
	}
}

func TestOptionOrderDiffersPerPlayerButTrueFalseIsFixed(t *testing.T) { // UT-QS-06
	var players []store.Player
	for i := 0; i < 15; i++ {
		players = append(players, player(fmt.Sprintf("p%02d", i)))
	}
	f := newPlay(t, players...)
	f.start()
	orders := map[string]bool{}
	for _, p := range players {
		shown := f.show(p.ID)
		key := ""
		for _, opt := range shown["options"].([]map[string]string) {
			key += opt["id"] + ","
		}
		orders[key] = true
		f.answer(p.ID, "M1-01", "opt-b")
		f.show(p.ID)
		f.answer(p.ID, "M1-02", "opt-b")
		tf := f.show(p.ID)["options"].([]map[string]string)
		if tf[0]["id"] != "opt-a" || tf[1]["id"] != "opt-b" {
			t.Fatalf("true/false options must keep their order, got %v", tf)
		}
	}
	if len(orders) < 2 {
		t.Error("15 players all saw the same option order")
	}
}

func TestLeaderboardIsThrottledToOncePerInterval(t *testing.T) {
	f := newPlayWithInterval(t, 300*time.Millisecond)
	f.start()
	f.show("p1")
	f.show("p2")
	f.host.drain()

	f.room.Answer("p1", "M1-01", strPtr("opt-b"))
	f.room.Answer("p2", "M1-01", strPtr("opt-b"))
	f.sinks["p1"].await(t, "q.result")
	f.sinks["p2"].await(t, "q.result")

	first := f.host.await(t, "lb.update").D.(map[string]any)
	if rows := first["rankings"].([]map[string]any); len(rows) != 2 {
		t.Fatalf("rankings = %v", rows)
	}
	f.host.expectQuiet(t) // the second answer must wait for the throttle window
	f.host.await(t, "lb.update")
	f.host.expectQuiet(t)
}

func TestLeaderboardUsesRedisOrderAndFallsBackToMemory(t *testing.T) {
	f := newPlay(t)
	f.start()
	f.show("p1")
	f.board.mu.Lock()
	f.board.order = []string{"p2", "p1"} // Redis says p2 leads even though nobody scored
	f.board.mu.Unlock()
	f.answer("p1", "M1-01", "opt-b")
	rows := f.host.await(t, "lb.update").D.(map[string]any)["rankings"].([]map[string]any)
	if rows[0]["nickname"] != "nick-p2" {
		t.Errorf("Redis order should be used, got %v", rows)
	}

	f.board.mu.Lock()
	f.board.orderOf = errors.New("redis down")
	saves := len(f.board.saved)
	f.board.mu.Unlock()
	f.show("p1")
	f.answer("p1", "M1-02", "opt-b")
	rows = f.host.await(t, "lb.update").D.(map[string]any)["rankings"].([]map[string]any)
	if rows[0]["nickname"] != "nick-p1" || rows[0]["score"] != 1300 {
		t.Errorf("memory ranking should take over, got %v", rows)
	}
	if f.board.saveCount() <= saves {
		t.Error("the leaderboard should be rebuilt in Redis after a failed read")
	}
}

func TestBoardIsRestoredFromRoomStateOnLoad(t *testing.T) {
	restored := player("p1")
	restored.Score, restored.CorrectCount = 900, 2
	f := newPlay(t, restored)
	f.board.mu.Lock()
	defer f.board.mu.Unlock()
	if len(f.board.saved) == 0 || f.board.saved[0][0].Score != 900 || f.board.saved[0][0].Correct != 2 {
		t.Errorf("standings not restored into Redis: %+v", f.board.saved)
	}
}

func TestSaveFailureKeepsStateAndAllowsRetry(t *testing.T) {
	f := newPlay(t)
	f.start()
	f.show("p1")
	f.st.mu.Lock()
	f.st.saveErr = errors.New("db down")
	f.st.mu.Unlock()

	f.room.Answer("p1", "M1-01", strPtr("opt-b"))
	f.sinks["p1"].awaitError(t, codeInternal)

	f.st.mu.Lock()
	f.st.saveErr = nil
	f.st.mu.Unlock()
	if result := f.answer("p1", "M1-01", "opt-b"); result["score"] != 500 {
		t.Errorf("retry should succeed, got %v", result)
	}
}

func TestDuplicateAnswerFromStoreIsReportedAsAnswered(t *testing.T) {
	f := newPlay(t)
	f.start()
	f.show("p1")
	f.st.mu.Lock()
	f.st.saveErr = store.ErrDuplicateAnswer
	f.st.mu.Unlock()
	f.room.Answer("p1", "M1-01", strPtr("opt-b"))
	f.sinks["p1"].awaitError(t, codeAnswered)
}

func TestKickingTheLastUnfinishedPlayerEndsTheRoom(t *testing.T) {
	f := newPlay(t)
	f.start()
	for _, q := range []string{"M1-01", "M1-02", "M1-03"} {
		f.show("p1")
		f.answer("p1", q, "opt-b")
	}
	f.host.drain()
	f.room.Kick("p2")
	f.host.await(t, "room.ended")
	if len(f.board.forgot) != 1 || f.board.forgot[0] != "p2" {
		t.Errorf("kicked player must leave the leaderboard, got %v", f.board.forgot)
	}
}

func TestFifteenPlayersAnsweringConcurrentlyIsRaceFree(t *testing.T) { // UT-RM-08
	var players []store.Player
	for i := 0; i < 15; i++ {
		players = append(players, player(fmt.Sprintf("p%02d", i)))
	}
	f := newPlay(t, players...)
	f.start()
	var wg sync.WaitGroup
	for _, p := range players {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, q := range []string{"M1-01", "M1-02", "M1-03"} {
				f.room.NextQuestion(p.ID)
				f.sinks[p.ID].await(t, "q.show")
				f.room.Answer(p.ID, q, strPtr("opt-b"))
				f.sinks[p.ID].await(t, "q.result")
			}
		}()
	}
	wg.Wait()
	f.host.await(t, "room.ended")
	if n := len(f.results()); n != 45 {
		t.Errorf("saved %d results, want 45", n)
	}
}

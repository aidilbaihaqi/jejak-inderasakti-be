package game

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/rank"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store"
)

// Error codes sent in {"t":"error"} messages; see contracts/ws.md.
const (
	codeNotStarted = "ROOM_NOT_STARTED"
	codeInProgress = "QUESTION_IN_PROGRESS"
	codeFinished   = "ALREADY_FINISHED"
	codeNotServed  = "NOT_SERVED"
	codeAnswered   = "ALREADY_ANSWERED"
	codeBadOption  = "INVALID_OPTION"
	codeInternal   = "INTERNAL"

	typeTrueFalse = "tf"
)

// activeQuestion is the question currently shown to a player.
type activeQuestion struct {
	question store.Question
	options  []store.Option // in the order this player sees them
	servedAt time.Time
	limit    time.Duration
}

// deadline is when an answer is still accepted: the timer plus the network tolerance.
func (a *activeQuestion) deadline() time.Time {
	return a.servedAt.Add(a.limit + networkGraceMs*time.Millisecond)
}

// outcome is how a question was resolved. A nil correct means it timed out.
type outcome struct {
	optionID   *string
	correct    *bool
	answeredAt *time.Time
	points     int
	spentMs    int
}

func (r *Room) NextQuestion(playerID string) {
	r.run(func() { r.nextQuestion(playerID) })
}

func (r *Room) Answer(playerID, questionID string, optionID *string) {
	r.run(func() { r.answer(playerID, questionID, optionID) })
}

func (r *Room) nextQuestion(playerID string) {
	p, ok := r.players[playerID]
	if !ok {
		return
	}
	if r.status != StatusRunning {
		r.reject(p, "q.next", codeNotStarted, "the session has not started")
		return
	}
	if p.active != nil && !r.settleExpired(p) {
		r.reject(p, "q.next", codeInProgress, "answer the current question first")
		return
	}
	if r.status != StatusRunning {
		return
	}
	if p.resolved >= len(r.questions) {
		r.reject(p, "q.next", codeFinished, "you have answered every question")
		return
	}
	r.serve(p)
}

func (r *Room) answer(playerID, questionID string, optionID *string) {
	p, ok := r.players[playerID]
	if !ok {
		return
	}
	if r.status != StatusRunning {
		r.reject(p, "q.answer", codeNotStarted, "the session has not started")
		return
	}
	a := p.active
	if a == nil || a.question.ID != questionID {
		r.rejectUnservedAnswer(p, questionID)
		return
	}
	if optionID != nil && !hasOption(a.options, *optionID) {
		r.reject(p, "q.answer", codeBadOption, "unknown option for this question")
		return
	}
	r.resolve(p, r.outcomeOf(p, a, optionID), true)
}

func (r *Room) rejectUnservedAnswer(p *playerState, questionID string) {
	if r.wasResolved(p, questionID) {
		r.reject(p, "q.answer", codeAnswered, "this question was already answered")
		return
	}
	r.reject(p, "q.answer", codeNotServed, "this question is not the active one")
}

func (r *Room) wasResolved(p *playerState, questionID string) bool {
	for _, q := range r.questions[:min(p.resolved, len(r.questions))] {
		if q.ID == questionID {
			return true
		}
	}
	return false
}

// outcomeOf scores an answer. A missing option or an answer past the deadline counts as a timeout.
func (r *Room) outcomeOf(p *playerState, a *activeQuestion, optionID *string) outcome {
	now := r.deps.clock()
	if optionID == nil || now.After(a.deadline()) {
		return timeoutOutcome(a)
	}
	elapsed := now.Sub(a.servedAt)
	correct := isCorrect(a.options, *optionID)
	points := Score(ScoreInput{
		Level: a.question.Level, Correct: correct, ResponseMs: elapsed.Milliseconds(), LimitMs: a.limit.Milliseconds(),
		GraceMs: readingGrace(r.jenjang), StreakBefore: p.streak, AccuracyMode: r.accuracyMode,
	})
	return outcome{optionID: optionID, correct: &correct, answeredAt: &now, points: points, spentMs: int(min(elapsed, a.limit).Milliseconds())}
}

func timeoutOutcome(a *activeQuestion) outcome {
	return outcome{spentMs: int(a.limit.Milliseconds())}
}

// resolve records the outcome of the active question, advances the player and updates rankings.
// It returns false, leaving the player untouched, if the result could not be persisted.
func (r *Room) resolve(p *playerState, out outcome, announce bool) bool {
	a := p.active
	correct := out.correct != nil && *out.correct
	next := *p
	next.score += out.points
	next.totalMs += out.spentMs
	next.resolved++
	next.streak, next.correct = 0, p.correct
	if correct {
		next.streak, next.correct = p.streak+1, p.correct+1
	}
	finished := next.resolved >= len(r.questions)

	if err := r.persist(p, a, out, next, finished); err != nil {
		r.rejectFailedSave(p, err)
		return false
	}
	p.score, p.totalMs, p.resolved, p.streak, p.correct, p.active = next.score, next.totalMs, next.resolved, next.streak, next.correct, nil
	r.saveStanding(p)
	if announce {
		r.sendResult(p, a.question, out, finished)
	}
	r.markLeaderboardDirty()
	if finished {
		r.endIfAllFinished()
	}
	return true
}

func (r *Room) persist(p *playerState, a *activeQuestion, out outcome, next playerState, finished bool) error {
	return r.withStoreTimeout(func(ctx context.Context) error {
		return r.deps.store.SaveResult(ctx, store.Result{
			PlayerID: p.info.ID, QuestionID: a.question.ID, ServedAt: a.servedAt,
			AnsweredAt: out.answeredAt, OptionID: out.optionID, Correct: out.correct, Points: out.points,
			Score: next.score, CorrectCount: next.correct, TotalMs: next.totalMs, CurrentIndex: next.resolved,
			Streak: next.streak, Finished: finished,
		})
	})
}

func (r *Room) rejectFailedSave(p *playerState, err error) {
	if errors.Is(err, store.ErrDuplicateAnswer) {
		r.reject(p, "q.answer", codeAnswered, "this question was already answered")
		return
	}
	slog.Error("save result", "room", r.id, "player", p.info.ID, "err", err)
	r.reject(p, "q.answer", codeInternal, "could not save the answer, try again")
}

func (r *Room) serve(p *playerState) {
	question := r.questions[p.resolved]
	p.active = &activeQuestion{
		question: question, options: r.shuffledOptions(question),
		servedAt: r.deps.clock(), limit: questionLimit(question.Level, r.jenjang),
	}
	r.sendQuestion(p, p.active.servedAt)
}

// sendQuestion shows the active question; limit_ms is the time left at now, which is less than the
// full limit only after a reconnect.
func (r *Room) sendQuestion(p *playerState, now time.Time) {
	a := p.active
	remaining := max(a.servedAt.Add(a.limit).Sub(now), 0)
	options := make([]map[string]string, len(a.options))
	for i, o := range a.options {
		options[i] = map[string]string{"id": o.ID, "label": o.Label.In(p.info.Lang)}
	}
	r.sendTo(p, Message{T: "q.show", D: map[string]any{
		"index": p.resolved, "total": len(r.questions), "site": a.question.Site, "level": a.question.Level,
		"prompt": a.question.Prompt.In(p.info.Lang), "options": options, "limit_ms": remaining.Milliseconds(),
	}})
}

func (r *Room) sendResult(p *playerState, q store.Question, out outcome, finished bool) {
	r.sendTo(p, Message{T: "q.result", D: map[string]any{
		"correct": out.correct != nil && *out.correct, "correct_option_id": correctOptionID(q.Options),
		"explanation": q.Explanation.In(p.info.Lang), "points": out.points,
		"score": p.score, "streak": p.streak, "finished": finished,
	}})
}

// settleExpired records a 0-point timeout if the active question is past its deadline.
// It reports whether the player is free to receive a new question.
func (r *Room) settleExpired(p *playerState) bool {
	if p.active == nil {
		return true
	}
	if r.status != StatusRunning || r.deps.clock().Before(p.active.deadline()) {
		return false
	}
	return r.resolve(p, timeoutOutcome(p.active), false)
}

// resumeActive re-sends the question a reconnecting player was working on.
func (r *Room) resumeActive(p *playerState) {
	if r.status == StatusRunning && p.active != nil {
		r.sendQuestion(p, r.deps.clock())
	}
}

func (r *Room) endIfAllFinished() {
	if r.status != StatusRunning || len(r.players) == 0 {
		return
	}
	for _, p := range r.players {
		if p.resolved < len(r.questions) {
			return
		}
	}
	r.endSession()
}

func (r *Room) shuffledOptions(q store.Question) []store.Option {
	options := append([]store.Option(nil), q.Options...)
	if q.Type != typeTrueFalse {
		r.rng.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })
	}
	return options
}

func (r *Room) reject(p *playerState, ref, code, message string) {
	r.sendTo(p, ErrorMessage(code, message, ref))
}

func (r *Room) sendTo(p *playerState, m Message) {
	if p.sink != nil {
		p.sink.Send(m)
	}
}

// --- leaderboard ---

func (r *Room) standings() []rank.Standing {
	standings := make([]rank.Standing, 0, len(r.order))
	for _, id := range r.order {
		standings = append(standings, standingOf(r.players[id]))
	}
	return standings
}

func standingOf(p *playerState) rank.Standing {
	return rank.Standing{PlayerID: p.info.ID, Score: p.score, Correct: p.correct, TotalMs: p.totalMs}
}

// markLeaderboardDirty sends lb.update at most once per interval, delaying it if needed.
func (r *Room) markLeaderboardDirty() {
	r.lbDirty = true
	if r.lbWaiting {
		return
	}
	wait := r.deps.lbInterval - time.Since(r.lbLast)
	if wait <= 0 {
		r.sendLeaderboard()
		return
	}
	r.lbWaiting = true
	r.lbTimer = time.AfterFunc(wait, func() {
		r.run(func() {
			r.lbWaiting = false
			if r.lbDirty && r.status != StatusEnded {
				r.sendLeaderboard()
			}
		})
	})
}

func (r *Room) sendLeaderboard() {
	rankings := make([]map[string]any, 0, len(r.players))
	for i, id := range r.rankedIDs() {
		p := r.players[id]
		rankings = append(rankings, map[string]any{
			"rank": i + 1, "nickname": p.info.Nickname, "school": p.info.School, "score": p.score, "correct_count": p.correct,
		})
	}
	r.broadcast(Message{T: "lb.update", D: map[string]any{"rankings": rankings}})
	r.lbDirty, r.lbLast = false, time.Now()
}

// rankedIDs asks Redis for the order and falls back to memory (rebuilding Redis) if it is missing or stale.
func (r *Room) rankedIDs() []string {
	standings := r.standings()
	if r.deps.board == nil {
		return rank.Order(standings)
	}
	ctx, cancel := context.WithTimeout(r.deps.ctx, boardTimeout)
	defer cancel()
	ids, err := r.deps.board.Order(ctx, r.id)
	if err == nil && r.coversAllPlayers(ids) {
		return ids
	}
	if err != nil {
		slog.Warn("read leaderboard, using memory", "room", r.id, "err", err)
	}
	r.saveStandings(standings)
	return rank.Order(standings)
}

func (r *Room) coversAllPlayers(ids []string) bool {
	if len(ids) != len(r.players) {
		return false
	}
	for _, id := range ids {
		if _, ok := r.players[id]; !ok {
			return false
		}
	}
	return true
}

func (r *Room) restoreBoard() {
	r.saveStandings(r.standings())
}

func (r *Room) saveStanding(p *playerState) {
	r.saveStandings([]rank.Standing{standingOf(p)})
}

func (r *Room) saveStandings(standings []rank.Standing) {
	if r.deps.board == nil {
		return
	}
	ctx, cancel := context.WithTimeout(r.deps.ctx, boardTimeout)
	defer cancel()
	if err := r.deps.board.Save(ctx, r.id, standings); err != nil {
		slog.Warn("save leaderboard", "room", r.id, "err", err)
	}
}

func (r *Room) forgetStanding(playerID string) {
	if r.deps.board == nil {
		return
	}
	ctx, cancel := context.WithTimeout(r.deps.ctx, boardTimeout)
	defer cancel()
	if err := r.deps.board.Forget(ctx, r.id, playerID); err != nil {
		slog.Warn("forget leaderboard entry", "room", r.id, "err", err)
	}
}

// --- option helpers ---

func hasOption(options []store.Option, id string) bool {
	for _, o := range options {
		if o.ID == id {
			return true
		}
	}
	return false
}

func isCorrect(options []store.Option, id string) bool {
	for _, o := range options {
		if o.ID == id {
			return o.Correct
		}
	}
	return false
}

func correctOptionID(options []store.Option) string {
	for _, o := range options {
		if o.Correct {
			return o.ID
		}
	}
	return ""
}

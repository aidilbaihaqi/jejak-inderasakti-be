package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store/queries"
)

var ErrDuplicateAnswer = errors.New("question already answered by this player")

// Text is bilingual content stored as {"id": "...", "en": "..."}.
type Text struct {
	ID string `json:"id"`
	EN string `json:"en"`
}

// In returns the text in the given language, falling back to Indonesian.
func (t Text) In(lang string) string {
	if lang == "en" && t.EN != "" {
		return t.EN
	}
	return t.ID
}

type Option struct {
	ID      string `json:"id"`
	Label   Text   `json:"label"`
	Correct bool   `json:"correct"`
}

type Question struct {
	ID          string
	Site, Level int
	Type        string
	Prompt      Text
	Explanation Text
	Options     []Option
}

// Result is one resolved question (answered or timed out) plus the player's new totals.
type Result struct {
	PlayerID, QuestionID string
	ServedAt             time.Time
	AnsweredAt           *time.Time // nil when the question timed out
	OptionID             *string
	Correct              *bool
	Points               int

	Score, CorrectCount, TotalMs, CurrentIndex, Streak int
	Finished                                           bool
}

// QuestionsByIDs returns the questions in the order of ids; a missing id is an error.
func (s *Store) QuestionsByIDs(ctx context.Context, ids []string) ([]Question, error) {
	rows, err := s.q.GetQuestionsByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get questions: %w", err)
	}
	byID := make(map[string]Question, len(rows))
	for _, row := range rows {
		question, err := questionFromRow(row)
		if err != nil {
			return nil, err
		}
		byID[question.ID] = question
	}
	ordered := make([]Question, 0, len(ids))
	for _, id := range ids {
		question, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("question %s: %w", id, ErrNotFound)
		}
		ordered = append(ordered, question)
	}
	return ordered, nil
}

func questionFromRow(row queries.GetQuestionsByIDsRow) (Question, error) {
	question := Question{ID: row.ID, Site: int(row.Site), Level: int(row.Level), Type: row.Type}
	if err := json.Unmarshal(row.Prompt, &question.Prompt); err != nil {
		return Question{}, fmt.Errorf("question %s prompt: %w", row.ID, err)
	}
	if err := json.Unmarshal(row.Explanation, &question.Explanation); err != nil {
		return Question{}, fmt.Errorf("question %s explanation: %w", row.ID, err)
	}
	if err := json.Unmarshal(row.Options, &question.Options); err != nil {
		return Question{}, fmt.Errorf("question %s options: %w", row.ID, err)
	}
	return question, nil
}

// SaveResult stores the answer and the player's new totals atomically.
// It returns ErrDuplicateAnswer if the player already has an answer for that question.
func (s *Store) SaveResult(ctx context.Context, r Result) error {
	playerID, err := uuid.Parse(r.PlayerID)
	if err != nil {
		return ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin save result: %w", err)
	}
	defer rollback(ctx, tx)

	q := s.q.WithTx(tx)
	inserted, err := q.InsertAnswer(ctx, answerParams(playerID, r))
	if err != nil {
		return fmt.Errorf("insert answer: %w", err)
	}
	if inserted == 0 {
		return ErrDuplicateAnswer
	}
	err = q.UpdatePlayerProgress(ctx, queries.UpdatePlayerProgressParams{
		ID: playerID, Score: toInt32(r.Score), CorrectCount: toInt16(r.CorrectCount), TotalMs: toInt32(r.TotalMs),
		CurrentIndex: toInt16(r.CurrentIndex), Streak: toInt16(r.Streak), Column7: r.Finished,
	})
	if err != nil {
		return fmt.Errorf("update player progress: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit save result: %w", err)
	}
	return nil
}

func answerParams(playerID uuid.UUID, r Result) queries.InsertAnswerParams {
	params := queries.InsertAnswerParams{
		RoomPlayerID: playerID, QuestionID: r.QuestionID, Points: toInt16(r.Points),
		ServedAt: pgtype.Timestamptz{Time: r.ServedAt, Valid: true},
	}
	if r.AnsweredAt != nil {
		params.AnsweredAt = pgtype.Timestamptz{Time: *r.AnsweredAt, Valid: true}
	}
	if r.OptionID != nil {
		params.OptionID = pgtype.Text{String: *r.OptionID, Valid: true}
	}
	if r.Correct != nil {
		params.Correct = pgtype.Bool{Bool: *r.Correct, Valid: true}
	}
	return params
}

// toInt16 and toInt32 clamp instead of wrapping; the database columns are smaller than int.
func toInt16(v int) int16 {
	return int16(min(max(v, math.MinInt16), math.MaxInt16)) //nolint:gosec // clamped to range above
}

func toInt32(v int) int32 {
	return int32(min(max(v, math.MinInt32), math.MaxInt32)) //nolint:gosec // clamped to range above
}

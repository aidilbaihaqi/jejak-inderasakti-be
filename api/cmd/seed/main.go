package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/config"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/seed"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store/queries"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("seed failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	questionsPath := flag.String("questions", "../seed/questions.json", "path to questions.json")
	schoolsPath := flag.String("schools", "../seed/schools.csv", "path to schools.csv")
	strict := flag.Bool("strict", false, "require the full 66-question launch bank")
	flag.Parse()

	questions, err := seed.LoadQuestions(*questionsPath)
	if err != nil {
		return err
	}
	if err := validate(questions, *strict); err != nil {
		return err
	}
	schools, err := seed.LoadSchools(*schoolsPath)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := store.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := store.Migrate(ctx, pool); err != nil {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer rollbackUnlessCommitted(ctx, tx)

	q := queries.New(tx)
	if err := upsertQuestions(ctx, q, questions); err != nil {
		return err
	}
	if err := upsertSchools(ctx, q, schools); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed: %w", err)
	}
	slog.Info("seed complete", "questions", len(questions), "schools", len(schools))
	return nil
}

func validate(questions []seed.Question, strict bool) error {
	if err := seed.ValidateQuestions(questions); err != nil {
		return fmt.Errorf("invalid questions: %w", err)
	}
	if !strict {
		return nil
	}
	if err := seed.ValidateBankSize(questions); err != nil {
		return fmt.Errorf("incomplete question bank: %w", err)
	}
	return nil
}

func upsertQuestions(ctx context.Context, q *queries.Queries, questions []seed.Question) error {
	for _, item := range questions {
		params, err := toQuestionParams(item)
		if err != nil {
			return err
		}
		if err := q.UpsertQuestion(ctx, params); err != nil {
			return fmt.Errorf("upsert question %s: %w", item.ID, err)
		}
	}
	return nil
}

func toQuestionParams(item seed.Question) (queries.UpsertQuestionParams, error) {
	prompt, err := json.Marshal(item.Prompt)
	if err != nil {
		return queries.UpsertQuestionParams{}, fmt.Errorf("marshal prompt %s: %w", item.ID, err)
	}
	options, err := json.Marshal(item.Options)
	if err != nil {
		return queries.UpsertQuestionParams{}, fmt.Errorf("marshal options %s: %w", item.ID, err)
	}
	explanation, err := json.Marshal(item.Explanation)
	if err != nil {
		return queries.UpsertQuestionParams{}, fmt.Errorf("marshal explanation %s: %w", item.ID, err)
	}
	return queries.UpsertQuestionParams{
		ID:          item.ID,
		Site:        int16(item.Site),
		Level:       int16(item.Level),
		Type:        item.Type,
		Prompt:      prompt,
		Options:     options,
		Explanation: explanation,
		Active:      item.IsActive(),
	}, nil
}

func upsertSchools(ctx context.Context, q *queries.Queries, schools []seed.School) error {
	for _, s := range schools {
		err := q.UpsertSchool(ctx, queries.UpsertSchoolParams{
			Name:    s.Name,
			Jenjang: textOrNull(s.Jenjang),
			City:    textOrNull(s.City),
		})
		if err != nil {
			return fmt.Errorf("upsert school %q: %w", s.Name, err)
		}
	}
	return nil
}

func rollbackUnlessCommitted(ctx context.Context, tx pgx.Tx) {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		slog.Warn("rollback seed transaction", "err", err)
	}
}

func textOrNull(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

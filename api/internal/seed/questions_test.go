package seed

import (
	"fmt"
	"strings"
	"testing"
)

func validQuestion() Question {
	return Question{
		ID: "M1-01", Site: 1, Level: 1, Type: "mc",
		Prompt:      Bilingual{ID: "Soal?", EN: "Question?"},
		Explanation: Bilingual{ID: "Penjelasan", EN: "Explanation"},
		Options: []Option{
			{ID: "opt-a", Label: Bilingual{ID: "a", EN: "a"}, Correct: true},
			{ID: "opt-b", Label: Bilingual{ID: "b", EN: "b"}},
		},
	}
}

func TestValidateQuestions(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Question)
		wantErr string
	}{
		{"valid", func(*Question) {}, ""},
		{"bad id format", func(q *Question) { q.ID = "X1-01" }, "id must look like"},
		{"level out of range", func(q *Question) { q.Level = 4 }, "level must be 1-3"},
		{"reserve id with site 3", func(q *Question) { q.ID = "X-01"; q.Site = 3 }, "reserve questions must have site 0"},
		{"reserve id valid", func(q *Question) { q.ID = "X-01"; q.Site = 0 }, ""},
		{"site mismatch", func(q *Question) { q.Site = 2 }, "does not match id"},
		{"missing english prompt", func(q *Question) { q.Prompt.EN = " " }, "prompt needs both"},
		{"missing explanation", func(q *Question) { q.Explanation.ID = "" }, "explanation needs both"},
		{"unknown type", func(q *Question) { q.Type = "essay" }, "unknown type"},
		{"no correct option", func(q *Question) { q.Options[0].Correct = false }, "exactly one option"},
		{"two correct options", func(q *Question) { q.Options[1].Correct = true }, "exactly one option"},
		{"duplicate option id", func(q *Question) { q.Options[1].ID = "opt-a" }, "invalid or duplicated"},
		{"tf with three options", func(q *Question) {
			q.Type = "tf"
			q.Options = append(q.Options, Option{ID: "opt-c", Label: Bilingual{ID: "c", EN: "c"}})
		}, "needs 2-2 options"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := validQuestion()
			tt.mutate(&q)
			assertErr(t, ValidateQuestions([]Question{q}), tt.wantErr)
		})
	}
}

func TestValidateQuestionsRejectsDuplicateIDs(t *testing.T) {
	err := ValidateQuestions([]Question{validQuestion(), validQuestion()})
	assertErr(t, err, "duplicate id")
}

func TestValidateBankSize(t *testing.T) {
	assertErr(t, ValidateBankSize([]Question{validQuestion()}), "expected 66 questions")

	full := make([]Question, 0, ExpectedQuestionCount)
	for i := 0; i < ExpectedQuestionCount; i++ {
		q := validQuestion()
		q.Site = i%siteCount + 1
		q.ID = fmt.Sprintf("M%d-%02d", q.Site, i)
		full = append(full, q)
	}
	assertErr(t, ValidateBankSize(full), "")
}

func TestQuestionIsActiveDefaultsToTrue(t *testing.T) {
	off := false
	if !(Question{}).IsActive() {
		t.Error("missing active should default to true")
	}
	if (Question{Active: &off}).IsActive() {
		t.Error("explicit false should stay inactive")
	}
}

func TestLoadQuestionsSeedFileIsValid(t *testing.T) {
	questions, err := LoadQuestions("../../../seed/questions.json")
	if err != nil {
		t.Fatal(err)
	}
	assertErr(t, ValidateQuestions(questions), "")
}

func TestSeedFileIsTheFullBank(t *testing.T) {
	questions, err := LoadQuestions("../../../seed/questions.json")
	if err != nil {
		t.Fatal(err)
	}
	assertErr(t, ValidateBankSize(questions), "")
}

func TestParseSchoolRows(t *testing.T) {
	got, err := parseSchoolRows([][]string{{" SDN 1 ", "SD", "Tanjungpinang"}, {"MAN 1", "", ""}})
	if err != nil || len(got) != 2 || got[0].Name != "SDN 1" || got[1].Jenjang != "" {
		t.Fatalf("unexpected result %+v, err %v", got, err)
	}
	if _, err := parseSchoolRows([][]string{{"", "SD", "x"}}); err == nil {
		t.Error("empty name should fail")
	}
	if _, err := parseSchoolRows([][]string{{"only-name"}}); err == nil {
		t.Error("short row should fail")
	}
}

func assertErr(t *testing.T, err error, want string) {
	t.Helper()
	switch {
	case want == "" && err != nil:
		t.Fatalf("unexpected error: %v", err)
	case want != "" && (err == nil || !strings.Contains(err.Error(), want)):
		t.Fatalf("want error containing %q, got %v", want, err)
	}
}

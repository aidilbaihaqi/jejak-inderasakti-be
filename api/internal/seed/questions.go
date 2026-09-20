package seed

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const (
	ExpectedQuestionCount = 66
	MinQuestionsPerSite   = 12
	siteCount             = 5
)

var (
	questionIDPattern = regexp.MustCompile(`^M([1-5])-\d{2}$`)
	reserveIDPattern  = regexp.MustCompile(`^X-\d{2}$`)
	optionIDPattern   = regexp.MustCompile(`^opt-[a-d]$`)
	optionCountByType = map[string][2]int{"mc": {2, 4}, "tf": {2, 2}, "fill": {2, 4}}
)

type Bilingual struct {
	ID string `json:"id"`
	EN string `json:"en"`
}

type Option struct {
	ID      string    `json:"id"`
	Label   Bilingual `json:"label"`
	Correct bool      `json:"correct"`
}

type Question struct {
	ID          string    `json:"id"`
	Site        int       `json:"site"`
	Level       int       `json:"level"`
	Type        string    `json:"type"`
	Prompt      Bilingual `json:"prompt"`
	Options     []Option  `json:"options"`
	Explanation Bilingual `json:"explanation"`
	Active      *bool     `json:"active"`
}

// IsActive treats a missing "active" field as true, matching the column default.
func (q Question) IsActive() bool {
	return q.Active == nil || *q.Active
}

func LoadQuestions(path string) ([]Question, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read questions file: %w", err)
	}
	var questions []Question
	if err := json.Unmarshal(raw, &questions); err != nil {
		return nil, fmt.Errorf("parse questions file: %w", err)
	}
	return questions, nil
}

// ValidateQuestions checks every question and returns all problems at once.
func ValidateQuestions(questions []Question) error {
	var problems []error
	seen := make(map[string]bool, len(questions))
	for _, q := range questions {
		if seen[q.ID] {
			problems = append(problems, fmt.Errorf("%s: duplicate id", q.ID))
		}
		seen[q.ID] = true
		problems = append(problems, validateQuestion(q)...)
	}
	return errors.Join(problems...)
}

// ValidateBankSize enforces the launch-day bank: 66 questions, at least 12 per site.
func ValidateBankSize(questions []Question) error {
	var problems []error
	if len(questions) != ExpectedQuestionCount {
		problems = append(problems, fmt.Errorf("expected %d questions, got %d", ExpectedQuestionCount, len(questions)))
	}
	perSite := make(map[int]int, siteCount)
	for _, q := range questions {
		perSite[q.Site]++
	}
	for site := 1; site <= siteCount; site++ {
		if perSite[site] < MinQuestionsPerSite {
			problems = append(problems, fmt.Errorf("site %d: expected at least %d questions, got %d", site, MinQuestionsPerSite, perSite[site]))
		}
	}
	return errors.Join(problems...)
}

func validateQuestion(q Question) []error {
	var problems []error
	fail := func(format string, args ...any) {
		problems = append(problems, fmt.Errorf("%s: %s", q.ID, fmt.Sprintf(format, args...)))
	}

	problems = append(problems, validateID(q)...)
	if isBlank(q.Prompt) {
		fail("prompt needs both id and en")
	}
	if isBlank(q.Explanation) {
		fail("explanation needs both id and en")
	}
	problems = append(problems, validateOptions(q)...)
	return problems
}

// reserveSite marks cross-site reserve questions (ids X-01..X-06).
const reserveSite = 0

func validateID(q Question) []error {
	if reserveIDPattern.MatchString(q.ID) {
		if q.Site != reserveSite {
			return []error{fmt.Errorf("%s: reserve questions must have site 0", q.ID)}
		}
		return nil
	}
	m := questionIDPattern.FindStringSubmatch(q.ID)
	if m == nil {
		return []error{fmt.Errorf("%s: id must look like M1-01 (site 1-5) or X-01 (reserve)", q.ID)}
	}
	var problems []error
	if m[1] != fmt.Sprint(q.Site) {
		problems = append(problems, fmt.Errorf("%s: site %d does not match id", q.ID, q.Site))
	}
	if q.Level < 1 || q.Level > 3 {
		problems = append(problems, fmt.Errorf("%s: level must be 1-3, got %d", q.ID, q.Level))
	}
	return problems
}

func validateOptions(q Question) []error {
	var problems []error
	fail := func(format string, args ...any) {
		problems = append(problems, fmt.Errorf("%s: %s", q.ID, fmt.Sprintf(format, args...)))
	}

	bounds, ok := optionCountByType[q.Type]
	if !ok {
		fail("unknown type %q", q.Type)
		return problems
	}
	if len(q.Options) < bounds[0] || len(q.Options) > bounds[1] {
		fail("type %s needs %d-%d options, got %d", q.Type, bounds[0], bounds[1], len(q.Options))
	}

	correct := 0
	ids := make(map[string]bool, len(q.Options))
	for _, o := range q.Options {
		if !optionIDPattern.MatchString(o.ID) || ids[o.ID] {
			fail("option id %q is invalid or duplicated", o.ID)
		}
		ids[o.ID] = true
		if isBlank(o.Label) {
			fail("option %s needs both id and en label", o.ID)
		}
		if o.Correct {
			correct++
		}
	}
	if correct != 1 {
		fail("exactly one option must be correct, got %d", correct)
	}
	return problems
}

func isBlank(b Bilingual) bool {
	return strings.TrimSpace(b.ID) == "" || strings.TrimSpace(b.EN) == ""
}

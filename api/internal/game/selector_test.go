package game

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store"
)

// bankPool mirrors seed/bank_soal.xlsx: per site 5 easy, 4 medium, 3 hard, plus reserves on site 0.
func bankPool() []store.QuestionRef {
	var pool []store.QuestionRef
	for site := 1; site <= 5; site++ {
		for level, count := range map[int]int{1: 5, 2: 4, 3: 3} {
			for i := 1; i <= count; i++ {
				pool = append(pool, store.QuestionRef{ID: fmt.Sprintf("M%d-L%d-%d", site, level, i), Site: site, Level: level})
			}
		}
	}
	return append(pool, store.QuestionRef{ID: "X-01", Site: 0, Level: 1})
}

func levelOf(t *testing.T, id string) int {
	t.Helper()
	var level int
	if _, err := fmt.Sscanf(id[strings.Index(id, "-L")+2:], "%d", &level); err != nil {
		t.Fatalf("parse level of %s: %v", id, err)
	}
	return level
}

func TestSelectQuestionsFollowsComposition(t *testing.T) {
	for jenjang, cells := range composition {
		t.Run(jenjang, func(t *testing.T) {
			ids, err := SelectQuestions(rand.New(rand.NewPCG(1, 2)), jenjang, false, bankPool())
			if err != nil {
				t.Fatal(err)
			}
			if len(ids) != stageCount*questionsEach {
				t.Fatalf("want 15 questions, got %d", len(ids))
			}
			for i, id := range ids {
				wantSite, wantLevel := i/questionsEach+1, cells[i/questionsEach][i%questionsEach]
				if !strings.HasPrefix(id, fmt.Sprintf("M%d-", wantSite)) || levelOf(t, id) != wantLevel {
					t.Errorf("position %d = %s, want site %d level %d", i, id, wantSite, wantLevel)
				}
			}
		})
	}
}

func TestSelectQuestionsShortSessionKeepsFirstAndLast(t *testing.T) {
	ids, err := SelectQuestions(rand.New(rand.NewPCG(3, 4)), "SMP", true, bankPool())
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 10 {
		t.Fatalf("want 10 questions, got %d", len(ids))
	}
	cells := composition["SMP"]
	for stage := 0; stage < stageCount; stage++ {
		if levelOf(t, ids[stage*2]) != cells[stage][0] || levelOf(t, ids[stage*2+1]) != cells[stage][2] {
			t.Errorf("stage %d levels wrong: %v", stage+1, ids[stage*2:stage*2+2])
		}
	}
}

func TestSelectQuestionsHasNoDuplicatesAndNoReserves(t *testing.T) {
	for seed := uint64(0); seed < 50; seed++ {
		ids, err := SelectQuestions(rand.New(rand.NewPCG(seed, seed)), "SD", false, bankPool())
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for _, id := range ids {
			if seen[id] || strings.HasPrefix(id, "X-") {
				t.Fatalf("seed %d: duplicate or reserve question %s in %v", seed, id, ids)
			}
			seen[id] = true
		}
	}
}

func TestSelectQuestionsDiffersBetweenRooms(t *testing.T) {
	first, _ := SelectQuestions(rand.New(rand.NewPCG(1, 1)), "SMA", false, bankPool())
	second, _ := SelectQuestions(rand.New(rand.NewPCG(99, 7)), "SMA", false, bankPool())
	if strings.Join(first, ",") == strings.Join(second, ",") {
		t.Error("different seeds produced the identical question set")
	}
}

func TestSelectQuestionsErrors(t *testing.T) {
	if _, err := SelectQuestions(rand.New(rand.NewPCG(1, 1)), "SMK", false, bankPool()); err == nil {
		t.Error("unknown jenjang should fail")
	}
	if _, err := SelectQuestions(rand.New(rand.NewPCG(1, 1)), "SD", false, nil); err == nil {
		t.Error("empty pool should fail")
	}
}

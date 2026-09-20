package rank

import (
	"reflect"
	"testing"
)

func TestOrderBreaksTiesByCorrectThenTime(t *testing.T) {
	standings := []Standing{
		{PlayerID: "slow", Score: 1000, Correct: 3, TotalMs: 40000},
		{PlayerID: "fast", Score: 1000, Correct: 3, TotalMs: 20000},
		{PlayerID: "more-correct", Score: 1000, Correct: 4, TotalMs: 90000},
		{PlayerID: "leader", Score: 2000, Correct: 1, TotalMs: 99000},
	}
	want := []string{"leader", "more-correct", "fast", "slow"}
	if got := Order(standings); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestOrderKeepsInputOrderForIdenticalStandings(t *testing.T) {
	standings := []Standing{{PlayerID: "a", Score: 500}, {PlayerID: "b", Score: 500}, {PlayerID: "c", Score: 500}}
	if got := Order(standings); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Errorf("got %v", got)
	}
}

func TestKeyIsExactAndMonotonicAtSessionLimits(t *testing.T) {
	best := Standing{Score: 14750, Correct: 15, TotalMs: 0}
	worse := Standing{Score: 14750, Correct: 15, TotalMs: 1}
	if !(best.Key() > worse.Key()) {
		t.Error("one millisecond faster must rank higher at the maximum score")
	}
	if !(Standing{Score: 1, Correct: 0, TotalMs: 469000}.Key() > Standing{Score: 0, Correct: 15, TotalMs: 0}.Key()) {
		t.Error("score must dominate correct count")
	}
}

func TestKeyClampsOutOfRangeTime(t *testing.T) {
	if (Standing{TotalMs: -5}).Key() != (Standing{TotalMs: 0}).Key() {
		t.Error("negative time should clamp to 0")
	}
	if (Standing{TotalMs: 5_000_000}).Key() != (Standing{TotalMs: maxTotalMs}).Key() {
		t.Error("huge time should clamp to the maximum")
	}
}

package game

import (
	"testing"
	"time"
)

func TestScore(t *testing.T) {
	medium := ScoreInput{Level: 2, Correct: true, LimitMs: 20000, GraceMs: 2000}
	with := func(base ScoreInput, edit func(*ScoreInput)) ScoreInput {
		edit(&base)
		return base
	}
	tests := []struct {
		id   string
		in   ScoreInput
		want int
	}{
		{"UT-SC-01 medium inside reading time", with(medium, func(i *ScoreInput) { i.ResponseMs = 1000 }), 750},
		{"UT-SC-02 medium fast", with(medium, func(i *ScoreInput) { i.ResponseMs = 4000 }), 717},
		{"UT-SC-03 medium average", with(medium, func(i *ScoreInput) { i.ResponseMs = 12000 }), 583},
		{"UT-SC-04 medium last second", with(medium, func(i *ScoreInput) { i.ResponseMs = 19000 }), 467},
		{"UT-SC-05 wrong answer", with(medium, func(i *ScoreInput) { i.Correct = false; i.ResponseMs = 1000 }), 0},
		{"UT-SC-06 past deadline plus grace", with(medium, func(i *ScoreInput) { i.ResponseMs = 21100 }), 0},
		{"UT-SC-07 easy inside reading time", ScoreInput{Level: 1, Correct: true, ResponseMs: 500, LimitMs: 15000, GraceMs: 2000}, 500},
		{"UT-SC-08 hard inside reading time", ScoreInput{Level: 3, Correct: true, ResponseMs: 500, LimitMs: 25000, GraceMs: 2000}, 1000},
		{"UT-SC-09 accuracy mode ignores speed", with(medium, func(i *ScoreInput) { i.AccuracyMode = true; i.ResponseMs = 10000 }), 750},
		{"UT-SC-10 second in a row", with(medium, func(i *ScoreInput) { i.ResponseMs = 1000; i.StreakBefore = 1 }), 800},
		{"UT-SC-11 third in a row", with(medium, func(i *ScoreInput) { i.ResponseMs = 1000; i.StreakBefore = 2 }), 850},
		{"UT-SC-12 streak bonus is capped", with(medium, func(i *ScoreInput) { i.ResponseMs = 1000; i.StreakBefore = 6 }), 1000},
		// SD: timer 20s*1.25 = 25s, reading time 3s. t=13s -> speed = 1-10/22 = 0.5455 -> 750*0.8182 = 613.6
		{"UT-SC-13 SD timer x1.25", ScoreInput{Level: 2, Correct: true, ResponseMs: 13000, LimitMs: 25000, GraceMs: 3000}, 614},
		{"UT-SC-14 SD reading time is 3s", ScoreInput{Level: 2, Correct: true, ResponseMs: 2500, LimitMs: 25000, GraceMs: 3000}, 750},
		{"answer inside the network grace still scores", with(medium, func(i *ScoreInput) { i.ResponseMs = 20500 }), 450},
		{"streak bonus also applies in accuracy mode", with(medium, func(i *ScoreInput) { i.AccuracyMode = true; i.StreakBefore = 3 }), 900},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			if got := Score(tt.in); got != tt.want {
				t.Errorf("Score() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestScoreNeverDividesByZero(t *testing.T) {
	in := ScoreInput{Level: 1, Correct: true, ResponseMs: 1000, LimitMs: 2000, GraceMs: 2000}
	if got := Score(in); got != 500 {
		t.Errorf("Score() = %d, want 500", got)
	}
}

func TestQuestionLimit(t *testing.T) {
	tests := []struct {
		level   int
		jenjang string
		want    time.Duration
	}{
		{1, "SMP", 15 * time.Second}, {2, "SMA", 20 * time.Second}, {3, "SMP", 25 * time.Second},
		{1, "SD", 18750 * time.Millisecond}, {2, "SD", 25 * time.Second}, {3, "SD", 31250 * time.Millisecond},
	}
	for _, tt := range tests {
		if got := questionLimit(tt.level, tt.jenjang); got != tt.want {
			t.Errorf("questionLimit(%d, %s) = %v, want %v", tt.level, tt.jenjang, got, tt.want)
		}
	}
	if readingGrace("SD") != 3000 || readingGrace("SMP") != 2000 {
		t.Error("reading time must be 3s for SD and 2s otherwise")
	}
}

package game

import (
	"math"
	"time"
)

const (
	networkGraceMs = 1000 // tolerance after the timer ends, see ADR-002
	streakBonus    = 50
	maxStreakBonus = 250
	speedShare     = 0.4 // share of the base points that depends on speed; the other 0.6 is fixed for a correct answer

	readingGraceMs   = 2000
	readingGraceSDMs = 3000
	smallSchoolTimer = 1.25 // SD timers are longer
	jenjangSD        = "SD"
)

var (
	basePoints   = map[int]float64{1: 500, 2: 750, 3: 1000}
	timerByLevel = map[int]time.Duration{1: 15 * time.Second, 2: 20 * time.Second, 3: 25 * time.Second}
)

type ScoreInput struct {
	Level        int
	Correct      bool
	ResponseMs   int64 // answered_at - served_at, measured by the server
	LimitMs      int64
	GraceMs      int64 // reading time: no penalty during it
	StreakBefore int   // correct answers in a row before this question
	AccuracyMode bool
}

// Score returns the points for one answer: round(base * (0.6 + 0.4*speed)) + streak bonus.
func Score(in ScoreInput) int {
	if !in.Correct || in.ResponseMs > in.LimitMs+networkGraceMs {
		return 0
	}
	points := int(math.Round(basePoints[in.Level] * (1 - speedShare + speedShare*speedFactor(in))))
	return points + min(streakBonus*in.StreakBefore, maxStreakBonus)
}

func speedFactor(in ScoreInput) float64 {
	if in.AccuracyMode || in.LimitMs <= in.GraceMs {
		return 1
	}
	responseMs := min(in.ResponseMs, in.LimitMs)
	effectiveMs := max(0, responseMs-in.GraceMs)
	return 1 - float64(effectiveMs)/float64(in.LimitMs-in.GraceMs)
}

// questionLimit is the answer timer for a level; SD rooms get 25% more time.
func questionLimit(level int, jenjang string) time.Duration {
	limit := timerByLevel[level]
	if jenjang == jenjangSD {
		return time.Duration(float64(limit) * smallSchoolTimer)
	}
	return limit
}

func readingGrace(jenjang string) int64 {
	if jenjang == jenjangSD {
		return readingGraceSDMs
	}
	return readingGraceMs
}

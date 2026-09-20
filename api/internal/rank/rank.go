// Package rank orders players: higher score first, then more correct answers, then less total time.
package rank

import "sort"

const (
	scoreUnit   = 100_000_000
	correctUnit = 1_000_000
	maxTotalMs  = correctUnit - 1
)

type Standing struct {
	PlayerID string
	Score    int
	Correct  int
	TotalMs  int
}

// Key folds the three ranking criteria into one number so a Redis sorted set can order players.
// It stays exact in float64 while Correct < 100 and TotalMs <= 999,999 (a session never exceeds ~470,000 ms).
func (s Standing) Key() float64 {
	ms := min(max(s.TotalMs, 0), maxTotalMs)
	return float64(s.Score)*scoreUnit + float64(s.Correct)*correctUnit + float64(maxTotalMs-ms)
}

// Order returns player ids best first. Equal standings keep their input order.
func Order(standings []Standing) []string {
	sorted := append([]Standing(nil), standings...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Key() > sorted[j].Key() })
	ids := make([]string, len(sorted))
	for i, s := range sorted {
		ids[i] = s.PlayerID
	}
	return ids
}

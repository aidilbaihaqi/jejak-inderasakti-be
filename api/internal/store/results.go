package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ResultRow is one player's final line in a room; equal standings share a rank.
type ResultRow struct {
	Rank                         int
	Nickname, School, Jenjang    string
	Score, CorrectCount, TotalMs int
}

// SchoolRank is a school's average of its top 5 scores across the event.
type SchoolRank struct {
	School      string
	AvgScore    float64
	PlayerCount int
}

// RoomResults returns every player of a room, best first (score, then correct answers, then time).
func (s *Store) RoomResults(ctx context.Context, roomID string) ([]ResultRow, error) {
	id, err := uuid.Parse(roomID)
	if err != nil {
		return nil, ErrNotFound
	}
	rows, err := s.q.GetRoomLeaderboard(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("room results: %w", err)
	}
	results := make([]ResultRow, 0, len(rows))
	for _, r := range rows {
		results = append(results, ResultRow{
			Rank: int(r.Rank), Nickname: r.Nickname, School: r.SchoolName, Jenjang: r.Jenjang,
			Score: int(r.Score), CorrectCount: int(r.CorrectCount), TotalMs: int(r.TotalMs),
		})
	}
	return results, nil
}

// SchoolLeaderboard ranks schools by the average of their 5 best finished players, minimum 3 players.
// The special "Sekolah lain" and "Umum" entries are not schools and are excluded.
func (s *Store) SchoolLeaderboard(ctx context.Context) ([]SchoolRank, error) {
	rows, err := s.q.GetSchoolLeaderboard(ctx, specialSchools)
	if err != nil {
		return nil, fmt.Errorf("school leaderboard: %w", err)
	}
	ranks := make([]SchoolRank, 0, len(rows))
	for _, r := range rows {
		ranks = append(ranks, SchoolRank{School: r.SchoolName, AvgScore: r.AvgScore, PlayerCount: int(r.PlayerCount)})
	}
	return ranks, nil
}
